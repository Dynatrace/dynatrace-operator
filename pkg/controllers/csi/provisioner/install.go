package csiprovisioner

import (
	"context"
	"encoding/base64"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/arch"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/symlink"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/url"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/processmoduleconfig"
)

func (provisioner *OneAgentProvisioner) installAgentImage(
	ctx context.Context,
	dk dynakube.DynaKube,
	latestProcessModuleConfig *dtclient.ProcessModuleConfig,
) (
	string,
	error,
) {
	targetImage := dk.CodeModulesImage()
	// An image URI often contains one or several /-s, which is problematic when trying to use it as a folder name.
	// Easiest to just base64 encode it
	base64Image := base64.StdEncoding.EncodeToString([]byte(targetImage))
	targetDir := provisioner.path.AgentSharedBinaryDirForAgent(base64Image)
	targetConfigDir := provisioner.path.AgentConfigDir(dk.GetName())

	props := &image.Properties{
		ImageUri:     targetImage,
		ApiReader:    provisioner.apiReader,
		Dynakube:     &dk,
		PathResolver: provisioner.path,
		Metadata:     provisioner.db,
	}

	imageInstaller, err := provisioner.imageInstallerBuilder(ctx, provisioner.fs, props)
	if err != nil {
		return "", err
	}

	err = provisioner.installAgent(ctx, imageInstaller, dk, targetDir, targetImage)
	if err != nil {
		return "", err
	}

	err = processmoduleconfig.CreateAgentConfigDir(provisioner.fs, targetConfigDir, targetDir, latestProcessModuleConfig)
	if err != nil {
		return "", err
	}

	return base64Image, err
}

func (provisioner *OneAgentProvisioner) installAgentZip(ctx context.Context, dk dynakube.DynaKube, dtc dtclient.Client, latestProcessModuleConfig *dtclient.ProcessModuleConfig) (string, error) {
	targetVersion := dk.CodeModulesVersion()
	urlInstaller := provisioner.urlInstallerBuilder(provisioner.fs, dtc, getUrlProperties(targetVersion, provisioner.path))

	targetDir := provisioner.path.AgentSharedBinaryDirForAgent(targetVersion)
	targetConfigDir := provisioner.path.AgentConfigDir(dk.GetName())

	err := provisioner.installAgent(ctx, urlInstaller, dk, targetDir, targetVersion)
	if err != nil {
		return "", err
	}

	err = processmoduleconfig.CreateAgentConfigDir(provisioner.fs, targetConfigDir, targetDir, latestProcessModuleConfig)
	if err != nil {
		return "", err
	}

	return targetVersion, nil
}

func (provisioner *OneAgentProvisioner) installAgent(ctx context.Context, agentInstaller installer.Installer, dk dynakube.DynaKube, targetDir, targetVersion string) error {
	eventRecorder := updaterEventRecorder{
		recorder: provisioner.recorder,
		dk:       &dk,
	}
	isNewlyInstalled, err := agentInstaller.InstallAgent(ctx, targetDir)

	if err != nil {
		eventRecorder.sendFailedInstallAgentVersionEvent(targetVersion, dk.GetName())

		return err
	}

	if isNewlyInstalled {
		eventRecorder.sendInstalledAgentVersionEvent(targetVersion, dk.GetName())
	}

	symlinkPath := provisioner.path.LatestAgentBinaryForDynaKube(dk.GetName())
	if err := symlink.Remove(provisioner.fs, symlinkPath); err != nil {
		return err
	}

	err = symlink.CreateForLatestVersion(provisioner.fs, dk, targetDir, symlinkPath)
	if err != nil {
		return err
	}

	return nil
}

func getUrlProperties(targetVersion string, pathResolver metadata.PathResolver) *url.Properties {
	return &url.Properties{
		Os:            dtclient.OsUnix,
		Type:          dtclient.InstallerTypePaaS,
		Arch:          arch.Arch,
		Flavor:        arch.Flavor,
		Technologies:  []string{"all"},
		TargetVersion: targetVersion,
		SkipMetadata:  true,
		PathResolver:  pathResolver,
	}
}

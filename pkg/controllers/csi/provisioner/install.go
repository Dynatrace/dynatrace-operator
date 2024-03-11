package csiprovisioner

import (
	"context"
	"strings"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/arch"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/url"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/processmoduleconfig"
)

func (provisioner *OneAgentProvisioner) installAgentImage(
	ctx context.Context,
	dynakube dynatracev1beta1.DynaKube,
	latestProcessModuleConfig *dtclient.ProcessModuleConfig,
) (
	string,
	error,
) {
	tenantUUID, err := dynakube.TenantUUIDFromApiUrl()
	if err != nil {
		return "", err
	}

	if err != nil {
		return "", err
	}

	targetImage := dynakube.CodeModulesImage()
	//imageDigest, err := provisioner.getDigest(ctx, dynakube, targetImage)

	imageVersion := strings.Split(targetImage, ":")[1]
	if err != nil {
		return "", err
	}

	props := &image.Properties{
		ImageUri:     targetImage,
		ApiReader:    provisioner.apiReader,
		Dynakube:     &dynakube,
		PathResolver: provisioner.path,
		Metadata:     provisioner.db,
	}

	imageInstaller, err := provisioner.imageInstallerBuilder(provisioner.fs, props)
	if err != nil {
		return "", err
	}

	targetDir := provisioner.path.AgentSharedBinaryDirForAgent(imageVersion)
	targetConfigDir := provisioner.path.AgentConfigDir(tenantUUID, dynakube.GetName())

	err = provisioner.installAgent(ctx, imageInstaller, dynakube, targetDir, targetImage, tenantUUID)
	if err != nil {
		return "", err
	}

	err = processmoduleconfig.CreateAgentConfigDir(provisioner.fs, targetConfigDir, targetDir, latestProcessModuleConfig)
	if err != nil {
		return "", err
	}

	return imageVersion /*what should we return here?*/, err
}

func (provisioner *OneAgentProvisioner) installAgentZip(ctx context.Context, dynakube dynatracev1beta1.DynaKube, dtc dtclient.Client, latestProcessModuleConfig *dtclient.ProcessModuleConfig) (string, error) {
	tenantUUID, err := dynakube.TenantUUIDFromApiUrl()
	if err != nil {
		return "", err
	}

	targetVersion := dynakube.CodeModulesVersion()
	urlInstaller := provisioner.urlInstallerBuilder(provisioner.fs, dtc, getUrlProperties(targetVersion, provisioner.path))

	targetDir := provisioner.path.AgentSharedBinaryDirForAgent(targetVersion)
	targetConfigDir := provisioner.path.AgentConfigDir(tenantUUID, dynakube.GetName())

	err = provisioner.installAgent(ctx, urlInstaller, dynakube, targetDir, targetVersion, tenantUUID)
	if err != nil {
		return "", err
	}

	err = processmoduleconfig.CreateAgentConfigDir(provisioner.fs, targetConfigDir, targetDir, latestProcessModuleConfig)
	if err != nil {
		return "", err
	}

	return targetVersion, nil
}

func (provisioner *OneAgentProvisioner) installAgent(ctx context.Context, agentInstaller installer.Installer, dynakube dynatracev1beta1.DynaKube, targetDir, targetVersion, tenantUUID string) error { //nolint: revive
	eventRecorder := updaterEventRecorder{
		recorder: provisioner.recorder,
		dynakube: &dynakube,
	}
	isNewlyInstalled, err := agentInstaller.InstallAgent(ctx, targetDir)

	if err != nil {
		eventRecorder.sendFailedInstallAgentVersionEvent(targetVersion, tenantUUID)

		return err
	}

	if isNewlyInstalled {
		eventRecorder.sendInstalledAgentVersionEvent(targetVersion, tenantUUID)
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

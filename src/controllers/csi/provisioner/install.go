package csiprovisioner

import (
	"context"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/arch"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/src/dockerconfig"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/installer"
	"github.com/Dynatrace/dynatrace-operator/src/installer/image"
	"github.com/Dynatrace/dynatrace-operator/src/installer/url"
	"github.com/Dynatrace/dynatrace-operator/src/processmoduleconfig"
	"github.com/spf13/afero"
)

func (provisioner *OneAgentProvisioner) installAgentImage(ctx context.Context, dynakube dynatracev1beta1.DynaKube, latestProcessModuleConfigCache *processModuleConfigCache) (string, error) {
	tenantUUID, err := dynakube.TenantUUIDFromApiUrl()
	if err != nil {
		return "", err
	}
	dockerConfig := dockerconfig.NewDockerConfig(provisioner.apiReader, dynakube)
	err = dockerConfig.StoreRequiredFiles(ctx, afero.Afero{Fs: provisioner.fs})
	if err != nil {
		return "", err
	}

	targetImage := dynakube.CodeModulesImage()
	imageDigest, err := image.GetDigest(targetImage)
	if err != nil {
		return "", err
	}

	imageInstaller := provisioner.imageInstallerBuilder(provisioner.fs, &image.Properties{
		ImageUri:     targetImage,
		PathResolver: provisioner.path,
		Metadata:     provisioner.db,
		DockerConfig: *dockerConfig,
		ImageDigest:  imageDigest,
	})
	if err != nil {
		return "", err
	}

	targetDir := provisioner.path.AgentSharedBinaryDirForAgent(imageDigest)
	targetConfigDir := provisioner.path.AgentConfigDir(tenantUUID)
	err = provisioner.installAgent(imageInstaller, dynakube, targetDir, targetImage, tenantUUID)
	if err != nil {
		return "", err
	}

	err = processmoduleconfig.CreateAgentConfigDir(provisioner.fs, targetConfigDir, targetDir, latestProcessModuleConfigCache.ProcessModuleConfig)
	if err != nil {
		return "", err
	}
	return imageDigest, err
}

func (provisioner *OneAgentProvisioner) installAgentZip(dynakube dynatracev1beta1.DynaKube, dtc dtclient.Client, latestProcessModuleConfigCache *processModuleConfigCache) (string, error) {
	tenantUUID, err := dynakube.TenantUUIDFromApiUrl()
	if err != nil {
		return "", err
	}
	targetVersion := dynakube.CodeModulesVersion()
	urlInstaller := provisioner.urlInstallerBuilder(provisioner.fs, dtc, getUrlProperties(targetVersion, provisioner.path))

	targetDir := provisioner.path.AgentSharedBinaryDirForAgent(targetVersion)
	targetConfigDir := provisioner.path.AgentConfigDir(tenantUUID)
	err = provisioner.installAgent(urlInstaller, dynakube, targetDir, targetVersion, tenantUUID)
	if err != nil {
		return "", err
	}

	err = processmoduleconfig.CreateAgentConfigDir(provisioner.fs, targetConfigDir, targetDir, latestProcessModuleConfigCache.ProcessModuleConfig)
	if err != nil {
		return "", err
	}
	return targetVersion, nil
}

func (provisioner *OneAgentProvisioner) installAgent(agentInstaller installer.Installer, dynakube dynatracev1beta1.DynaKube, targetDir, targetVersion, tenantUUID string) error {
	defer agentInstaller.Cleanup()
	eventRecorder := updaterEventRecorder{
		recorder: provisioner.recorder,
		dynakube: &dynakube,
	}
	isNewlyInstalled, err := agentInstaller.InstallAgent(targetDir)
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

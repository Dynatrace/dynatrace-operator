package csiprovisioner

import (
	"context"
	"encoding/base64"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/arch"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	csiotel "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/internal/otel"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/url"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/processmoduleconfig"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/dtotel"
)

func (provisioner *OneAgentProvisioner) installAgentImage(
	ctx context.Context,
	dk dynakube.DynaKube,
	latestProcessModuleConfig *dtclient.ProcessModuleConfig,
) (
	targetImage string,
	err error,
) {
	tenantUUID, err := dk.TenantUUIDFromConnectionInfoStatus()
	if err != nil {
		return "", err
	}

	targetImage = dk.CodeModulesImage()
	// An image URI often contains one or several /-s, which is problematic when trying to use it as a folder name.
	// Easiest to just base64 encode it
	base64Image := base64.StdEncoding.EncodeToString([]byte(targetImage))
	targetDir := provisioner.path.AgentSharedBinaryDirForAgent(base64Image)
	targetConfigDir := provisioner.path.AgentConfigDir(tenantUUID, dk.GetName())

	defer func() {
		if err == nil {
			err = processmoduleconfig.CreateAgentConfigDir(provisioner.fs, targetConfigDir, targetDir, latestProcessModuleConfig)
		}
	}()

	codeModule, err := provisioner.db.ReadCodeModule(metadata.CodeModule{Version: targetImage})
	if codeModule != nil {
		log.Info("target image already downloaded", "image", targetImage, "path", targetDir)

		return targetImage, nil
	}

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

	ctx, span := dtotel.StartSpan(ctx, csiotel.Tracer(), csiotel.SpanOptions()...)
	defer span.End()

	err = provisioner.installAgent(ctx, imageInstaller, dk, targetDir, targetImage, tenantUUID)
	if err != nil {
		span.RecordError(err)

		return "", err
	}

	err = provisioner.db.CreateCodeModule(&metadata.CodeModule{
		Version:  targetImage,
		Location: targetDir,
	})
	if err != nil {
		return "", err
	}

	return targetImage, err
}

func (provisioner *OneAgentProvisioner) installAgentZip(ctx context.Context, dk dynakube.DynaKube, dtc dtclient.Client, latestProcessModuleConfig *dtclient.ProcessModuleConfig) (string, error) {
	tenantUUID, err := dk.TenantUUIDFromConnectionInfoStatus()
	if err != nil {
		return "", err
	}

	targetVersion := dk.CodeModulesVersion()
	urlInstaller := provisioner.urlInstallerBuilder(provisioner.fs, dtc, getUrlProperties(targetVersion, provisioner.path))

	targetDir := provisioner.path.AgentSharedBinaryDirForAgent(targetVersion)
	targetConfigDir := provisioner.path.AgentConfigDir(tenantUUID, dk.GetName())

	defer func() {
		if err == nil {
			err = processmoduleconfig.CreateAgentConfigDir(provisioner.fs, targetConfigDir, targetDir, latestProcessModuleConfig)
		}
	}()

	codeModule, err := provisioner.db.ReadCodeModule(metadata.CodeModule{Version: targetVersion})
	if codeModule != nil {
		log.Info("target version already downloaded", "version", targetVersion, "path", targetDir)

		return targetVersion, nil
	}

	ctx, span := dtotel.StartSpan(ctx, csiotel.Tracer(), csiotel.SpanOptions()...)
	defer span.End()

	err = provisioner.installAgent(ctx, urlInstaller, dk, targetDir, targetVersion, tenantUUID)
	if err != nil {
		span.RecordError(err)

		return "", err
	}

	err = provisioner.db.CreateCodeModule(&metadata.CodeModule{
		Version:  targetVersion,
		Location: targetDir,
	})
	if err != nil {
		return "", err
	}

	return targetVersion, nil
}

func (provisioner *OneAgentProvisioner) installAgent(ctx context.Context, agentInstaller installer.Installer, dk dynakube.DynaKube, targetDir, targetVersion, tenantUUID string) error { //nolint: revive
	eventRecorder := updaterEventRecorder{
		recorder: provisioner.recorder,
		dk:       &dk,
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

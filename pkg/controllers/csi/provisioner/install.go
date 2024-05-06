package csiprovisioner

import (
	"context"
	"encoding/base64"

	dynatracev1beta2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
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
	dynakube dynatracev1beta2.DynaKube,
	latestProcessModuleConfig *dtclient.ProcessModuleConfig,
) (
	string,
	error,
) {
	tenantUUID, err := dynakube.TenantUUIDFromConnectionInfoStatus()
	if err != nil {
		return "", err
	}

	targetImage := dynakube.CodeModulesImage()

	props := &image.Properties{
		ImageUri:     targetImage,
		ApiReader:    provisioner.apiReader,
		Dynakube:     &dynakube,
		PathResolver: provisioner.path,
		Metadata:     provisioner.db,
	}

	imageInstaller, err := provisioner.imageInstallerBuilder(ctx, provisioner.fs, props)
	if err != nil {
		return "", err
	}

	// An image URI often contains one or several /-s, which is problematic when trying to use it as a folder name.
	// Easiest to just base64 encode it
	targetDir := provisioner.path.AgentSharedBinaryDirForAgent(base64.StdEncoding.EncodeToString([]byte(targetImage)))
	targetConfigDir := provisioner.path.AgentConfigDir(tenantUUID, dynakube.GetName())

	ctx, span := dtotel.StartSpan(ctx, csiotel.Tracer(), csiotel.SpanOptions()...)
	defer span.End()

	err = provisioner.installAgent(ctx, imageInstaller, dynakube, targetDir, targetImage, tenantUUID)
	if err != nil {
		span.RecordError(err)

		return "", err
	}

	err = processmoduleconfig.CreateAgentConfigDir(provisioner.fs, targetConfigDir, targetDir, latestProcessModuleConfig)
	if err != nil {
		return "", err
	}

	return targetImage, err
}

func (provisioner *OneAgentProvisioner) installAgentZip(ctx context.Context, dynakube dynatracev1beta2.DynaKube, dtc dtclient.Client, latestProcessModuleConfig *dtclient.ProcessModuleConfig) (string, error) {
	tenantUUID, err := dynakube.TenantUUIDFromConnectionInfoStatus()
	if err != nil {
		return "", err
	}

	targetVersion := dynakube.CodeModulesVersion()
	urlInstaller := provisioner.urlInstallerBuilder(provisioner.fs, dtc, getUrlProperties(targetVersion, provisioner.path))

	targetDir := provisioner.path.AgentSharedBinaryDirForAgent(targetVersion)
	targetConfigDir := provisioner.path.AgentConfigDir(tenantUUID, dynakube.GetName())

	ctx, span := dtotel.StartSpan(ctx, csiotel.Tracer(), csiotel.SpanOptions()...)
	defer span.End()

	err = provisioner.installAgent(ctx, urlInstaller, dynakube, targetDir, targetVersion, tenantUUID)
	if err != nil {
		span.RecordError(err)

		return "", err
	}

	err = processmoduleconfig.CreateAgentConfigDir(provisioner.fs, targetConfigDir, targetDir, latestProcessModuleConfig)
	if err != nil {
		return "", err
	}

	return targetVersion, nil
}

func (provisioner *OneAgentProvisioner) installAgent(ctx context.Context, agentInstaller installer.Installer, dynakube dynatracev1beta2.DynaKube, targetDir, targetVersion, tenantUUID string) error { //nolint: revive
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

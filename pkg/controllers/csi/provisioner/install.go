package csiprovisioner

import (
	"context"
	"net/http"
	"strings"

	dynatracev1beta2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/arch"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	csiotel "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/internal/otel"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/url"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/processmoduleconfig"
	"github.com/Dynatrace/dynatrace-operator/pkg/oci/registry"
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
	tenantUUID, err := dynakube.TenantUUIDFromConnectionInfo()
	if err != nil {
		return "", err
	}

	targetImage := dynakube.CodeModulesImage()
	imageDigest, err := provisioner.getDigest(ctx, dynakube, targetImage)

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

	targetDir := provisioner.path.AgentSharedBinaryDirForAgent(imageDigest)
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

	return imageDigest, err
}

func (provisioner *OneAgentProvisioner) getDigest(ctx context.Context, dynakube dynatracev1beta2.DynaKube, imageUri string) (string, error) {
	pullSecret := dynakube.PullSecretWithoutData()
	defaultTransport := http.DefaultTransport.(*http.Transport).Clone()

	transport, err := registry.PrepareTransportForDynaKube(ctx, provisioner.apiReader, defaultTransport, &dynakube)
	if err != nil {
		return "", err
	}

	registryClient, err := provisioner.registryClientBuilder(
		registry.WithContext(ctx),
		registry.WithApiReader(provisioner.apiReader),
		registry.WithKeyChainSecret(&pullSecret),
		registry.WithTransport(transport),
	)
	if err != nil {
		return "", err
	}

	imageVersion, err := registryClient.GetImageVersion(ctx, imageUri)
	if err != nil {
		return "", err
	}

	digest, _ := strings.CutPrefix(string(imageVersion.Digest), "sha256:")

	return digest, nil
}

func (provisioner *OneAgentProvisioner) installAgentZip(ctx context.Context, dynakube dynatracev1beta2.DynaKube, dtc dtclient.Client, latestProcessModuleConfig *dtclient.ProcessModuleConfig) (string, error) {
	tenantUUID, err := dynakube.TenantUUIDFromConnectionInfo()
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

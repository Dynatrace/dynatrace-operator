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
	"github.com/spf13/afero"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type agentUpdater struct {
	fs            afero.Fs
	path          metadata.PathResolver
	targetDir     string
	targetVersion string
	tenantUUID    string
	installer     installer.Installer
	recorder      updaterEventRecorder
}

func newAgentUrlUpdater(fs afero.Fs, dtc dtclient.Client, previousVersion string, path metadata.PathResolver, recorder record.EventRecorder, dk *dynatracev1beta1.DynaKube) (*agentUpdater, error) { //nolint:revive // argument-limit doesn't apply to constructors
	tenantUUID, err := dk.TenantUUIDFromApiUrl()
	if err != nil {
		return nil, err
	}
	targetVersion := dk.CodeModulesVersion()

	agentInstaller := url.NewInstaller(fs, dtc, getUrlProperties(targetVersion, previousVersion, path))
	eventRecorder := updaterEventRecorder{
		recorder: recorder,
		dynakube: dk,
	}
	return &agentUpdater{
		fs:            fs,
		path:          path,
		targetDir:     path.AgentBinaryDirForVersion(tenantUUID, targetVersion),
		targetVersion: targetVersion,
		tenantUUID:    tenantUUID,
		installer:     agentInstaller,
		recorder:      eventRecorder,
	}, nil
}

func newAgentImageUpdater( //nolint:revive // argument-limit doesn't apply to constructors
	ctx context.Context,
	fs afero.Fs,
	apiReader client.Reader,
	path metadata.PathResolver,
	db metadata.Access,
	recorder record.EventRecorder,
	dk *dynatracev1beta1.DynaKube) (*agentUpdater, error) {
	tenantUUID, err := dk.TenantUUIDFromApiUrl()
	if err != nil {
		return nil, err
	}

	agentInstaller, err := newImageInstaller(ctx, fs, apiReader, path, db, dk)
	if err != nil {
		return nil, err
	}

	eventRecorder := updaterEventRecorder{
		recorder: recorder,
		dynakube: dk,
	}

	return &agentUpdater{
		fs:            fs,
		path:          path,
		targetDir:     path.AgentConfigDir(tenantUUID),
		targetVersion: dk.CodeModulesVersion(),
		tenantUUID:    tenantUUID,
		installer:     agentInstaller,
		recorder:      eventRecorder,
	}, nil
}

func getUrlProperties(targetVersion, previousVersion string, pathResolver metadata.PathResolver) *url.Properties {
	return &url.Properties{
		Os:              dtclient.OsUnix,
		Type:            dtclient.InstallerTypePaaS,
		Arch:            arch.Arch,
		Flavor:          arch.Flavor,
		Technologies:    []string{"all"},
		PreviousVersion: previousVersion,
		TargetVersion:   targetVersion,
		PathResolver:    pathResolver,
	}
}

func newImageInstaller(ctx context.Context, fs afero.Fs, apiReader client.Reader, pathResolver metadata.PathResolver, db metadata.Access, dynakube *dynatracev1beta1.DynaKube) (installer.Installer, error) { //nolint:revive // argument-limit doesn't apply to constructors
	dockerConfig := dockerconfig.NewDockerConfig(apiReader, *dynakube)
	if dynakube.Spec.CustomPullSecret != "" {
		err := dockerConfig.StoreRequiredFiles(ctx, afero.Afero{Fs: fs})
		if err != nil {
			return nil, err
		}
	}

	imageInstaller := image.NewImageInstaller(fs, &image.Properties{
		ImageUri:     dynakube.CodeModulesImage(),
		PathResolver: pathResolver,
		Metadata:     db,
		DockerConfig: *dockerConfig})
	return imageInstaller, nil
}

func (updater *agentUpdater) updateAgent(latestProcessModuleConfigCache *processModuleConfigCache) (string, error) {
	defer func(installer installer.Installer) {
		_ = installer.Cleanup()
	}(updater.installer)
	var updatedVersion string

	log.Info("updating agent",
		"target version", updater.targetVersion,
		"target directory", updater.targetDir,
	)

	err := updater.installAgent()
	if err != nil {
		return "", err
	}
	updatedVersion = updater.targetVersion

	log.Info("updating ruxitagentproc.conf on latest installed version")
	if err := updater.installer.UpdateProcessModuleConfig(updater.targetDir, latestProcessModuleConfigCache.ProcessModuleConfig); err != nil {
		return "", err
	}
	return updatedVersion, nil
}

func (updater *agentUpdater) installAgent() error {
	isNewlyInstalled, err := updater.installer.InstallAgent(updater.targetDir)
	if err != nil {
		updater.recorder.sendFailedInstallAgentVersionEvent(updater.targetVersion, updater.tenantUUID)
		return err
	}
	if isNewlyInstalled {
		updater.recorder.sendInstalledAgentVersionEvent(updater.targetVersion, updater.tenantUUID)
	}
	return nil
}

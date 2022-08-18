package csiprovisioner

import (
	"context"
	"os"

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

func newAgentUrlUpdater(
	ctx context.Context,
	fs afero.Fs,
	dtc dtclient.Client,
	previousVersion string,
	path metadata.PathResolver,
	recorder record.EventRecorder,
	dk *dynatracev1beta1.DynaKube) (*agentUpdater, error) {

	tenantUUID := dk.ConnectionInfo().TenantUUID
	targetVersion := dk.CodeModulesVersion()

	agentInstaller := url.NewUrlInstaller(fs, dtc, getUrlProperties(targetVersion, previousVersion, path))
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

func newAgentImageUpdater(
	ctx context.Context,
	fs afero.Fs,
	apiReader client.Reader,
	path metadata.PathResolver,
	db metadata.Access,
	recorder record.EventRecorder,
	dk *dynatracev1beta1.DynaKube) (*agentUpdater, error) {

	tenantUUID := dk.ConnectionInfo().TenantUUID
	certPath := path.ImageCertPath(tenantUUID)

	agentInstaller, err := setupImageInstaller(ctx, fs, apiReader, path, db, certPath, dk)
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

func setupImageInstaller(ctx context.Context, fs afero.Fs, apiReader client.Reader, pathResolver metadata.PathResolver, db metadata.Access, certPath string, dynakube *dynatracev1beta1.DynaKube) (installer.Installer, error) {
	dockerConfig := dockerconfig.NewDockerConfig(apiReader, *dynakube)
	if dynakube.Spec.CustomPullSecret != "" {
		err := dockerConfig.SetupAuths(ctx)
		if err != nil {
			return nil, err
		}
	}

	if dynakube.Spec.TrustedCAs != "" {
		err := dockerConfig.SaveCustomCAs(ctx, afero.Afero{Fs: fs}, certPath)
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
	defer updater.cleanCertsIfPresent()
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

func (updater *agentUpdater) cleanCertsIfPresent() {
	err := updater.fs.RemoveAll(updater.path.ImageCertPath(updater.tenantUUID))
	if err != nil && os.IsNotExist(err) {
		log.Info("no ca.crt found to clean")
	} else if err != nil {
		log.Error(err, "failed to clean ca.crt")
	}
}

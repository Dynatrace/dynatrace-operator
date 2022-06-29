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
	path metadata.PathResolver,
	recorder record.EventRecorder,
	dk *dynatracev1beta1.DynaKube) (*agentUpdater, error) {

	tenantUUID := dk.ConnectionInfo().TenantUUID
	targetVersion := dk.CodeModulesVersion()

	agentInstaller := url.NewUrlInstaller(fs, dtc, getUrlProperties(targetVersion))
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
	recorder record.EventRecorder,
	dk *dynatracev1beta1.DynaKube,
	digest string) (*agentUpdater, error) {

	tenantUUID := dk.ConnectionInfo().TenantUUID
	certPath := path.ImageCertPath(tenantUUID)

	agentInstaller, err := setupImageInstaller(ctx, fs, path, apiReader, certPath, digest, dk)
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

func getUrlProperties(version string) *url.Properties {
	return &url.Properties{
		Os:           dtclient.OsUnix,
		Type:         dtclient.InstallerTypePaaS,
		Arch:         arch.Arch,
		Flavor:       arch.Flavor,
		Technologies: []string{"all"},
		Version:      version,
	}
}

func setupImageInstaller(ctx context.Context, fs afero.Fs, pathResolver metadata.PathResolver, apiReader client.Reader, certPath, digest string, dynakube *dynatracev1beta1.DynaKube) (installer.Installer, error) {
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
		ImageDigest:  digest,
		PathResolver: pathResolver,
		DockerConfig: *dockerConfig})
	return imageInstaller, nil
}

func (updater *agentUpdater) updateAgent(installedVersion string, latestProcessModuleConfigCache *processModuleConfigCache) (string, error) {
	defer updater.cleanCertsIfPresent()
	var updatedVersion string
	if updater.versionDirNotPresent() || installedVersion == "" {
		log.Info("updating agent",
			"target version", updater.targetVersion,
			"installed version", installedVersion,
			"target directory", updater.targetDir)
		if err := updater.installer.InstallAgent(updater.targetDir); err != nil {
			updater.recorder.sendFailedInstallAgentVersionEvent(updater.targetVersion, updater.tenantUUID)
			return "", err
		}
		updater.recorder.sendInstalledAgentVersionEvent(updater.targetVersion, updater.tenantUUID)
		updatedVersion = updater.targetVersion
	}
	log.Info("updating ruxitagentproc.conf on latest installed version")
	if err := updater.installer.UpdateProcessModuleConfig(updater.targetDir, latestProcessModuleConfigCache.ProcessModuleConfig); err != nil {
		return "", err
	}
	return updatedVersion, nil
}

func (updater *agentUpdater) cleanCertsIfPresent() {
	err := updater.fs.RemoveAll(updater.path.ImageCertPath(updater.tenantUUID))
	if err != nil && os.IsNotExist(err) {
		log.Info("no ca.crt found to clean")
	} else if err != nil {
		log.Error(err, "failed to clean ca.crt")
	}
}

func (updater agentUpdater) versionDirNotPresent() bool {
	_, err := updater.fs.Stat(updater.targetDir)
	return os.IsNotExist(err)
}

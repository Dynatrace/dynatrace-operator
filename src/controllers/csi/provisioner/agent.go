package csiprovisioner

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/csi/provisioner/arch"
	"github.com/Dynatrace/dynatrace-operator/src/dockerconfig"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/installer"
	"github.com/Dynatrace/dynatrace-operator/src/installer/image"
	"github.com/Dynatrace/dynatrace-operator/src/installer/url"
	"github.com/spf13/afero"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type agentUpdater struct {
	fs        afero.Fs
	dk        *dynatracev1beta1.DynaKube
	path      metadata.PathResolver
	dtc       dtclient.Client
	installer installer.Installer
	apiReader client.Reader
	recorder  record.EventRecorder
}

var installUrlProperties = url.Properties{
	Os:           dtclient.OsUnix,
	Type:         dtclient.InstallerTypePaaS,
	Arch:         arch.Arch,
	Flavor:       arch.Flavor,
	Technologies: []string{"all"},
}

func newAgentUpdater(
	apiReader client.Reader,
	dtc dtclient.Client,
	path metadata.PathResolver,
	fs afero.Fs,
	recorder record.EventRecorder,
	dk *dynatracev1beta1.DynaKube,
) *agentUpdater {
	return &agentUpdater{
		fs:        fs,
		path:      path,
		apiReader: apiReader,
		recorder:  recorder,
		dk:        dk,
		dtc:       dtc,
	}
}

func (updater *agentUpdater) updateAgent(ctx context.Context, latestVersion, tenantUUID string, latestProcessModuleConfigCache *processModuleConfigCache) (string, error) {
	dk := updater.dk
	targetVersion := updater.getOneAgentVersionFromInstance()
	targetDir := updater.path.AgentBinaryDirForVersion(tenantUUID, targetVersion)

	var updatedVersion string
	if _, err := updater.fs.Stat(targetDir); os.IsNotExist(err) {
		log.Info("updating agent",
			"target version", targetVersion,
			"installed version", latestVersion,
			"target directory", targetDir)
		_ = updater.fs.MkdirAll(targetDir, 0755)
		if dk.CodeModulesImage() != "" {
			cleanCerts, err := updater.initImageInstaller(ctx, targetDir)
			if err != nil {
				return "", err
			}
			if cleanCerts != nil {
				defer cleanCerts()
			}
		} else {
			updater.initUrlInstaller()
		}

		if err := updater.installer.InstallAgent(targetDir); err != nil {
			updater.sendFailedInstallAgentVersionEvent(targetVersion, tenantUUID)
			return "", err
		}
		updater.sendInstalledAgentVersionEvent(targetVersion, tenantUUID)
		updatedVersion = targetVersion
	}
	log.Info("updating ruxitagentproc.conf on latest installed version")
	if err := updater.installer.UpdateProcessModuleConfig(targetDir, latestProcessModuleConfigCache.ProcessModuleConfig); err != nil {
		return "", err
	}

	return updatedVersion, nil
}

func (updater *agentUpdater) initUrlInstaller() {
	// To allow mocking
	if updater.installer == nil {
		updater.installer = url.NewUrlInstaller(updater.fs, updater.dtc, &installUrlProperties)
	}
}

func (updater *agentUpdater) initImageInstaller(ctx context.Context, targetDir string) (func(), error) {
	var cleanCerts func()
	dk := updater.dk
	dockerConfig, err := dockerconfig.NewDockerConfig(ctx, updater.apiReader, *dk)
	if err != nil {
		return cleanCerts, err
	}
	if dk.Spec.TrustedCAs != "" {
		caCertPath := filepath.Join(targetDir, "ca.crt")
		err := dockerConfig.SaveCustomCAs(ctx, afero.Afero{Fs: updater.fs}, caCertPath)
		if err != nil {
			return cleanCerts, err
		}
		cleanCerts = func() {
			if err := updater.fs.RemoveAll(caCertPath); err != nil {
				log.Error(err, "failed to remove ca.crt")
			}
		}
	}
	// To allow mocking
	if updater.installer == nil {
		updater.installer = image.NewImageInstaller(updater.fs, &image.Properties{
			ImageUri:     dk.CodeModulesImage(),
			DockerConfig: *dockerConfig,
		})
	}
	return cleanCerts, nil
}

func (updater *agentUpdater) getOneAgentVersionFromInstance() string {
	dk := updater.dk
	if dk.CodeModulesImage() != "" {
		image := dk.CodeModulesImage()
		return strings.Split(image, ":")[1]
	}
	if dk.Version() != "" {
		return dk.Version()
	}
	return dk.Status.LatestAgentVersionUnixPaas
}

func (updater *agentUpdater) sendFailedInstallAgentVersionEvent(version, tenantUUID string) {
	updater.recorder.Eventf(updater.dk,
		corev1.EventTypeWarning,
		failedInstallAgentVersionEvent,
		"Failed to install agent version: %s to tenant: %s", version, tenantUUID)
}

func (updater *agentUpdater) sendInstalledAgentVersionEvent(version, tenantUUID string) {
	updater.recorder.Eventf(updater.dk,
		corev1.EventTypeNormal,
		installAgentVersionEvent,
		"Installed agent version: %s to tenant: %s", version, tenantUUID)
}

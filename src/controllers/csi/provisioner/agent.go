package csiprovisioner

import (
	"context"
	"os"
	"path/filepath"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/csi/provisioner/arch"
	"github.com/Dynatrace/dynatrace-operator/src/dockerconfig"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/installer"
	"github.com/spf13/afero"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type agentUpdater struct {
	fs        afero.Fs
	dk        *dynatracev1beta1.DynaKube
	path      metadata.PathResolver
	installer installer.Installer
	apiReader client.Reader
	recorder  record.EventRecorder
}

func newAgentUpdater(
	apiReader client.Reader,
	dtc dtclient.Client,
	path metadata.PathResolver,
	fs afero.Fs,
	recorder record.EventRecorder,
	dk *dynatracev1beta1.DynaKube,
) *agentUpdater {
	agentInstaller := installer.NewOneAgentInstaller(
		fs,
		dtc,
		installer.InstallerProperties{
			Os:           dtclient.OsUnix,
			Type:         dtclient.InstallerTypePaaS,
			Arch:         arch.Arch,
			Flavor:       arch.Flavor,
			Technologies: []string{"all"},
		},
	)
	return &agentUpdater{
		fs:        fs,
		path:      path,
		apiReader: apiReader,
		recorder:  recorder,
		dk:        dk,
		installer: agentInstaller,
	}
}

func (updater *agentUpdater) updateAgent(ctx context.Context, installedVersion, tenantUUID string, previousHash string, latestProcessModuleConfigCache *processModuleConfigCache) (string, error) {
	dk := updater.dk
	targetVersion := updater.getOneAgentVersionFromInstance()
	targetDir := updater.path.AgentBinaryDirForVersion(tenantUUID, targetVersion)

	if _, err := updater.fs.Stat(targetDir); os.IsNotExist(err) {
		log.Info("updating agent",
			"target version", targetVersion,
			"installed version", installedVersion,
			"target directory", targetDir)

		if dk.CodeModulesImage() != "" { // TODO: Make this nicer
			dockerConfig, err := dockerconfig.NewDockerConfig(ctx, updater.apiReader, *dk)
			if err != nil {
				return "", err
			}
			if dk.Spec.TrustedCAs != "" {
				caCertPath := filepath.Join(targetDir, "ca.crt")
				err := dockerConfig.SaveCustomCAs(ctx, updater.apiReader, *dk, caCertPath)
				defer func() { updater.fs.Remove(caCertPath) }()
				if err != nil {
					return "", err
				}
			}
			updater.installer.SetImagePullInfo(&installer.ImagePullInfo{
				Image:        dk.CodeModulesImage(),
				DockerConfig: *dockerConfig,
			})
		}
		updater.installer.SetVersion(targetVersion)
		if err := updater.installer.InstallAgent(targetDir); err != nil {
			updater.recorder.Eventf(dk,
				corev1.EventTypeWarning,
				failedInstallAgentVersionEvent,
				"Failed to install agent version: %s to tenant: %s, err: %s", targetVersion, tenantUUID, err)
			return "", err
		}
		log.Info("updating ruxitagentproc.conf on new version")
		if err := updater.installer.UpdateProcessModuleConfig(targetDir, latestProcessModuleConfigCache.ProcessModuleConfig); err != nil {
			return "", err
		}
		updater.recorder.Eventf(dk,
			corev1.EventTypeNormal,
			installAgentVersionEvent,
			"Installed agent version: %s to tenant: %s", targetVersion, tenantUUID)
		return targetVersion, nil
	}
	if latestProcessModuleConfigCache != nil && previousHash != latestProcessModuleConfigCache.Hash {
		log.Info("updating ruxitagentproc.conf on latest installed version")
		if err := updater.installer.UpdateProcessModuleConfig(targetDir, latestProcessModuleConfigCache.ProcessModuleConfig); err != nil {
			return "", err
		}
	}

	return "", nil
}

func (updater *agentUpdater) getOneAgentVersionFromInstance() string {
	dk := updater.dk
	currentVersion := dk.Status.LatestAgentVersionUnixPaas
	if dk.Version() != "" {
		currentVersion = dk.Version()
	}
	return currentVersion
}

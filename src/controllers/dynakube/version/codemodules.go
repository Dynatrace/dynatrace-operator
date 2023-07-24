package version

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/src/api/status"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/dockerconfig"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
)

type codeModulesUpdater struct {
	dynakube *dynatracev1beta1.DynaKube
	dtClient dtclient.Client
}

func newCodeModulesUpdater(dynakube *dynatracev1beta1.DynaKube, dtClient dtclient.Client) *codeModulesUpdater {
	return &codeModulesUpdater{
		dynakube: dynakube,
		dtClient: dtClient,
	}
}

func (updater codeModulesUpdater) Name() string {
	return "codemodules"
}

func (updater codeModulesUpdater) IsEnabled() bool {
	return updater.dynakube.NeedAppInjection()
}

func (updater *codeModulesUpdater) Target() *status.VersionStatus {
	return &updater.dynakube.Status.CodeModules.VersionStatus
}

func (updater codeModulesUpdater) CustomImage() string {
	return updater.dynakube.CustomCodeModulesImage()
}

func (updater codeModulesUpdater) CustomVersion() string {
	return updater.dynakube.CustomCodeModulesVersion()
}

func (updater codeModulesUpdater) IsAutoUpdateEnabled() bool {
	return true
}

func (updater codeModulesUpdater) IsPublicRegistryEnabled() bool {
	return updater.dynakube.FeaturePublicRegistry()
}

func (updater codeModulesUpdater) LatestImageInfo() (*dtclient.LatestImageInfo, error) {
	return updater.dtClient.GetLatestCodeModulesImage()
}

func (updater *codeModulesUpdater) CheckForDowngrade(latestVersion string) (bool, error) {
	return false, nil
}

func (updater *codeModulesUpdater) UseTenantRegistry(_ context.Context, _ *dockerconfig.DockerConfig) error {
	customVersion := updater.CustomVersion()
	if customVersion != "" {
		updater.dynakube.Status.CodeModules = dynatracev1beta1.CodeModulesStatus{
			VersionStatus: status.VersionStatus{
				Version: customVersion,
			},
		}
		return nil
	}

	latestAgentVersionUnixPaas, err := updater.dtClient.GetLatestAgentVersion(
		dtclient.OsUnix, dtclient.InstallerTypePaaS)
	if err != nil {
		log.Info("could not get agent paas unix version")
		return err
	}

	updater.dynakube.Status.CodeModules = dynatracev1beta1.CodeModulesStatus{
		VersionStatus: status.VersionStatus{
			Version: latestAgentVersionUnixPaas,
		},
	}
	return nil
}

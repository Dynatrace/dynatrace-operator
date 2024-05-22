package version

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	dynatracev1beta2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"k8s.io/apimachinery/pkg/api/meta"
)

const (
	cmConditionType = "CodeModulesVersion"
)

type codeModulesUpdater struct {
	dynakube *dynatracev1beta2.DynaKube
	dtClient dtclient.Client
}

func newCodeModulesUpdater(dynakube *dynatracev1beta2.DynaKube, dtClient dtclient.Client) *codeModulesUpdater {
	return &codeModulesUpdater{
		dynakube: dynakube,
		dtClient: dtClient,
	}
}

func (updater codeModulesUpdater) Name() string {
	return "codemodules"
}

func (updater codeModulesUpdater) IsEnabled() bool {
	if updater.dynakube.NeedAppInjection() {
		return true
	}

	updater.dynakube.Status.CodeModules.VersionStatus = status.VersionStatus{}
	_ = meta.RemoveStatusCondition(updater.dynakube.Conditions(), cmConditionType)

	return false
}

func (updater *codeModulesUpdater) Target() *status.VersionStatus {
	return &updater.dynakube.Status.CodeModules.VersionStatus
}

func (updater codeModulesUpdater) CustomImage() string {
	customImage := updater.dynakube.CustomCodeModulesImage()
	if customImage != "" {
		setVerificationSkippedReasonCondition(updater.dynakube.Conditions(), cmConditionType)
	}

	return customImage
}

func (updater codeModulesUpdater) CustomVersion() string {
	return updater.dynakube.CustomCodeModulesVersion()
}

func (updater codeModulesUpdater) IsAutoUpdateEnabled() bool {
	return true
}

func (updater codeModulesUpdater) IsPublicRegistryEnabled() bool {
	isPublicRegistry := updater.dynakube.FeaturePublicRegistry()
	if isPublicRegistry {
		setVerifiedCondition(updater.dynakube.Conditions(), cmConditionType) // Bit hacky, as things can still go wrong, but if so we will just overwrite this is LatestImageInfo.
	}

	return isPublicRegistry
}

func (updater codeModulesUpdater) LatestImageInfo(ctx context.Context) (*dtclient.LatestImageInfo, error) {
	imgInfo, err := updater.dtClient.GetLatestCodeModulesImage(ctx)
	if err != nil {
		conditions.SetDynatraceApiError(updater.dynakube.Conditions(), cmConditionType, err)
	}

	return imgInfo, err
}

func (updater *codeModulesUpdater) CheckForDowngrade(_ string) (bool, error) {
	return false, nil
}

func (updater *codeModulesUpdater) UseTenantRegistry(ctx context.Context) error {
	customVersion := updater.CustomVersion()
	if customVersion != "" {
		updater.dynakube.Status.CodeModules = dynatracev1beta2.CodeModulesStatus{
			VersionStatus: status.VersionStatus{
				Version: customVersion,
			},
		}
		setVerificationSkippedReasonCondition(updater.dynakube.Conditions(), cmConditionType)

		return nil
	}

	latestAgentVersionUnixPaas, err := updater.dtClient.GetLatestAgentVersion(ctx,
		dtclient.OsUnix, dtclient.InstallerTypePaaS)
	if err != nil {
		log.Info("could not get agent paas unix version")
		conditions.SetDynatraceApiError(updater.dynakube.Conditions(), cmConditionType, err)

		return err
	}

	updater.dynakube.Status.CodeModules = dynatracev1beta2.CodeModulesStatus{
		VersionStatus: status.VersionStatus{
			Version: latestAgentVersionUnixPaas,
		},
	}
	setVerifiedCondition(updater.dynakube.Conditions(), cmConditionType)

	return nil
}

func (updater codeModulesUpdater) ValidateStatus() error {
	return nil
}

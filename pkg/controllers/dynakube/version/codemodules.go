package version

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	"k8s.io/apimachinery/pkg/api/meta"
)

const (
	cmConditionType = "CodeModulesVersion"
)

type codeModulesUpdater struct {
	dk       *dynakube.DynaKube
	dtClient dtclient.Client
}

func newCodeModulesUpdater(dk *dynakube.DynaKube, dtClient dtclient.Client) *codeModulesUpdater {
	return &codeModulesUpdater{
		dk:       dk,
		dtClient: dtClient,
	}
}

func (updater codeModulesUpdater) Name() string {
	return "codemodules"
}

func (updater codeModulesUpdater) IsEnabled() bool {
	if updater.dk.OneAgent().IsAppInjectionNeeded() {
		return true
	}

	updater.dk.Status.CodeModules.VersionStatus = status.VersionStatus{}
	_ = meta.RemoveStatusCondition(updater.dk.Conditions(), cmConditionType)

	return false
}

func (updater *codeModulesUpdater) Target() *status.VersionStatus {
	return &updater.dk.Status.CodeModules.VersionStatus
}

func (updater codeModulesUpdater) CustomImage() string {
	customImage := updater.dk.OneAgent().GetCustomCodeModulesImage()
	if customImage != "" {
		setVerificationSkippedReasonCondition(updater.dk.Conditions(), cmConditionType)
	}

	return customImage
}

func (updater codeModulesUpdater) CustomVersion() string {
	return updater.dk.OneAgent().GetCustomCodeModulesVersion()
}

func (updater codeModulesUpdater) IsAutoUpdateEnabled() bool {
	return true
}

func (updater codeModulesUpdater) IsPublicRegistryEnabled() bool {
	isPublicRegistry := updater.dk.FF().IsPublicRegistry()
	if isPublicRegistry {
		setVerifiedCondition(updater.dk.Conditions(), cmConditionType) // Bit hacky, as things can still go wrong, but if so we will just overwrite this is LatestImageInfo.
	}

	return isPublicRegistry
}

func (updater codeModulesUpdater) LatestImageInfo(ctx context.Context) (*dtclient.LatestImageInfo, error) {
	imgInfo, err := updater.dtClient.GetLatestCodeModulesImage(ctx)
	if err != nil {
		k8sconditions.SetDynatraceAPIError(updater.dk.Conditions(), cmConditionType, err)
	}

	return imgInfo, err
}

func (updater *codeModulesUpdater) CheckForDowngrade(_ string) (bool, error) {
	return false, nil
}

func (updater *codeModulesUpdater) UseTenantRegistry(ctx context.Context) error {
	customVersion := updater.CustomVersion()
	if customVersion != "" {
		updater.dk.Status.CodeModules = oneagent.CodeModulesStatus{
			VersionStatus: status.VersionStatus{
				Version: customVersion,
			},
		}
		setVerificationSkippedReasonCondition(updater.dk.Conditions(), cmConditionType)

		return nil
	}

	latestAgentVersionUnixPaas, err := updater.dtClient.GetLatestAgentVersion(ctx,
		dtclient.OsUnix, dtclient.InstallerTypePaaS)
	if err != nil {
		log.Info("could not get agent paas unix version")
		k8sconditions.SetDynatraceAPIError(updater.dk.Conditions(), cmConditionType, err)

		return err
	}

	updater.dk.Status.CodeModules = oneagent.CodeModulesStatus{
		VersionStatus: status.VersionStatus{
			Version: latestAgentVersionUnixPaas,
		},
	}
	setVerifiedCondition(updater.dk.Conditions(), cmConditionType)

	return nil
}

func (updater codeModulesUpdater) ValidateStatus() error {
	return nil
}

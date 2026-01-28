package kspmsettings

import (
	"context"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	dtsettings "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/settings"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/meta"
)

var (
	log = logd.Get().WithName("kspm-settings")
)

type Reconciler struct {
	timeProvider *timeprovider.Provider
}

func NewReconciler() *Reconciler {
	return &Reconciler{
		timeProvider: timeprovider.New(),
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, dtc dtsettings.APIClient, dk *dynakube.DynaKube) error {
	// Kubernetes Monitoring is REQUIRED for KSPM, so it is ok to just check for this.
	if !dk.ActiveGate().IsKubernetesMonitoringEnabled() {
		_ = meta.RemoveStatusCondition(dk.Conditions(), conditionType)

		return nil
	}

	if !k8sconditions.IsOutdated(r.timeProvider, dk, conditionType) {
		return nil
	}

	_ = meta.RemoveStatusCondition(dk.Conditions(), conditionType) // needed so the timestamp updates, will never actually show up in the status

	hasReadScope := k8sconditions.IsOptionalScopeAvailable(dk, dtclient.ConditionTypeAPITokenSettingsRead)
	hasWriteScope := k8sconditions.IsOptionalScopeAvailable(dk, dtclient.ConditionTypeAPITokenSettingsWrite)

	var missingScopes []string
	if !hasReadScope {
		missingScopes = append(missingScopes, dtclient.TokenScopeSettingsRead)
	}

	if !hasWriteScope {
		missingScopes = append(missingScopes, dtclient.TokenScopeSettingsWrite)
	}

	if len(missingScopes) > 0 {
		message := strings.Join(missingScopes, ", ") + " scope(s) missing: cannot query existing kspm monitoring setting and/or safely create new one."
		k8sconditions.SetOptionalScopeMissing(dk.Conditions(), conditionType, message)
		log.Info(message)

		return nil
	} else {
		log.Info("necessary scopes for kspm settings creation is available, proceeding with reconciliation")
	}

	err := r.checkKSPMSettings(ctx, dtc, dk)
	if err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) checkKSPMSettings(ctx context.Context, dtc dtsettings.APIClient, dk *dynakube.DynaKube) error {
	log.Info("start reconciling kspm settings")

	if dk.Status.KubernetesClusterMEID == "" {
		msg := "kubernetesClusterMEID is not available, which is needed for kspm settings creation, will skip it for now"
		log.Info(msg)

		setSkippedCondition(dk.Conditions(), msg)

		return nil
	}

	kspmSettings, err := dtc.GetKSPMSettings(ctx, dk.Status.KubernetesClusterMEID)
	if err != nil {
		setErrorCondition(dk.Conditions(), err.Error())

		return errors.WithMessage(err, "error trying to check if setting exists")
	}

	if kspmSettings.TotalCount > 0 {
		log.Info("there are already settings", "settings", kspmSettings)

		setExistsCondition(dk.Conditions())

		return nil
	}

	datasetPipelineEnabled := dk.KSPM().IsEnabled()

	objectID, err := dtc.CreateKSPMSetting(ctx, dk.Status.KubernetesClusterMEID, datasetPipelineEnabled)
	if err != nil {
		setErrorCondition(dk.Conditions(), err.Error())

		return err
	}

	setExistsCondition(dk.Conditions())
	log.Info("kspm setting created", "settings", objectID, "configurationDatasetPipelineEnabled", datasetPipelineEnabled)

	return nil
}

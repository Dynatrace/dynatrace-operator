package kspmsettings

import (
	"context"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	dtsettings "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/settings"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/tenant/optionalscopes"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/meta"
)

type Reconciler struct {
	timeProvider *timeprovider.Provider
}

func NewReconciler() *Reconciler {
	return &Reconciler{
		timeProvider: timeprovider.New(),
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, dtClient dtsettings.Client, dk *dynakube.DynaKube) error {
	ctx, log := logd.NewFromContext(ctx, "kspm-settings")
	// Kubernetes Monitoring is REQUIRED for KSPM, so it is ok to just check for this.
	if !dk.ActiveGate().IsKubernetesMonitoringEnabled() {
		_ = meta.RemoveStatusCondition(dk.Conditions(), conditionType)

		return nil
	}

	if !k8sconditions.IsOutdated(r.timeProvider, dk, conditionType) {
		return nil
	}

	_ = meta.RemoveStatusCondition(dk.Conditions(), conditionType) // needed so the timestamp updates, will never actually show up in the status

	hasReadScope := optionalscopes.IsAvailable(dk.OptionalScopes(), token.ScopeSettingsRead)
	hasWriteScope := optionalscopes.IsAvailable(dk.OptionalScopes(), token.ScopeSettingsWrite)

	var missingScopes []string
	if !hasReadScope {
		missingScopes = append(missingScopes, token.ScopeSettingsRead)
	}

	if !hasWriteScope {
		missingScopes = append(missingScopes, token.ScopeSettingsWrite)
	}

	if len(missingScopes) > 0 {
		message := strings.Join(missingScopes, ", ") + " scope(s) missing: cannot query existing kspm monitoring setting and/or safely create new one."
		k8sconditions.SetOptionalScopeMissing(dk.Conditions(), conditionType, message)
		log.Info(message)

		return nil
	} else {
		log.Info("necessary scopes for kspm settings creation is available, proceeding with reconciliation")
	}

	err := r.checkKSPMSettings(ctx, dtClient, dk)
	if err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) checkKSPMSettings(ctx context.Context, dtClient dtsettings.Client, dk *dynakube.DynaKube) error {
	log := logd.FromContext(ctx)
	log.Info("start reconciling kspm settings")

	if dk.Status.KubernetesClusterMEID == "" {
		msg := "kubernetesClusterMEID is not available, which is needed for kspm settings creation, will skip it for now"
		log.Info(msg)

		setSkippedCondition(dk.Conditions(), msg)

		return nil
	}

	kspmSettings, err := dtClient.GetKSPMSettings(ctx, dk.Status.KubernetesClusterMEID)
	if err != nil {
		if core.IsForbidden(err) {
			log.Info("tenant requires additional scopes for getting KSPM settings. Skipping reconciliation")

			return nil
		}

		setErrorCondition(dk.Conditions())

		return errors.WithMessage(err, "error trying to check if setting exists")
	}

	if kspmSettings.TotalCount > 0 {
		log.Info("there are already settings", "settings", kspmSettings)

		setExistsCondition(dk.Conditions())

		return nil
	}

	datasetPipelineEnabled := dk.KSPM().IsEnabled()

	objectID, err := dtClient.CreateKSPMSetting(ctx, dk.Status.KubernetesClusterMEID, datasetPipelineEnabled)
	if err != nil {
		if core.IsForbidden(err) {
			log.Info("tenant requires additional scopes for creating KSPM settings. Skipping reconciliation")

			return nil
		}

		setErrorCondition(dk.Conditions())

		return err
	}

	setExistsCondition(dk.Conditions())
	log.Info("kspm setting created", "settings", objectID, "configurationDatasetPipelineEnabled", datasetPipelineEnabled)

	return nil
}

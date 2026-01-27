package logmonsettings

import (
	"context"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/logmonitoring"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/settings"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/meta"
)

type Reconciler struct {
	dk  *dynakube.DynaKube
	dtc settings.APIClient

	timeProvider *timeprovider.Provider
}

func NewReconciler(dtc settings.APIClient, dk *dynakube.DynaKube) *Reconciler {
	return &Reconciler{
		dk:           dk,
		dtc:          dtc,
		timeProvider: timeprovider.New(),
	}
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	if !r.dk.LogMonitoring().IsEnabled() {
		_ = meta.RemoveStatusCondition(r.dk.Conditions(), ConditionType)

		return nil
	}

	if !k8sconditions.IsOutdated(r.timeProvider, r.dk, ConditionType) {
		return nil
	}

	_ = meta.RemoveStatusCondition(r.dk.Conditions(), ConditionType)

	hasReadScope := k8sconditions.IsOptionalScopeAvailable(r.dk, dtclient.ConditionTypeAPITokenSettingsRead)
	hasWriteScope := k8sconditions.IsOptionalScopeAvailable(r.dk, dtclient.ConditionTypeAPITokenSettingsWrite)

	var missingScopes []string
	if !hasReadScope {
		missingScopes = append(missingScopes, dtclient.TokenScopeSettingsRead)
	}

	if !hasWriteScope {
		missingScopes = append(missingScopes, dtclient.TokenScopeSettingsWrite)
	}

	if len(missingScopes) > 0 {
		message := strings.Join(missingScopes, ", ") + " scope(s) missing: cannot query existing log monitoring setting and/or safely create new one."
		k8sconditions.SetOptionalScopeMissing(r.dk.Conditions(), ConditionType, message)
		log.Info(message)

		return nil
	} else {
		log.Info("necessary scopes for logmonitoring settings creation is available, proceeding with reconciliation")
	}

	err := r.checkLogMonitoringSettings(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) checkLogMonitoringSettings(ctx context.Context) error {
	log.Info("start reconciling log monitoring settings")

	if r.dk.Status.KubernetesClusterMEID == "" {
		msg := "kubernetesClusterMEID is not available, which is needed for logmonitoring settings creation, will skip it for now"
		log.Info(msg)

		setSkippedCondition(r.dk.Conditions(), msg)

		return nil
	}

	logMonitoringSettings, err := r.dtc.GetSettingsForLogModule(ctx, r.dk.Status.KubernetesClusterMEID)
	if err != nil {
		setErrorCondition(r.dk.Conditions(), err.Error())

		return errors.WithMessage(err, "error trying to check if setting exists")
	}

	if logMonitoringSettings.TotalCount > 0 {
		log.Info("there are already settings", "settings", logMonitoringSettings)

		setExistsCondition(r.dk.Conditions())

		return nil
	}

	matchers := []logmonitoring.IngestRuleMatchers{}
	if len(r.dk.LogMonitoring().IngestRuleMatchers) > 0 {
		matchers = r.dk.LogMonitoring().IngestRuleMatchers
	}

	objectID, err := r.dtc.CreateLogMonitoringSetting(ctx, r.dk.Status.KubernetesClusterMEID, r.dk.Status.KubernetesClusterName, matchers)
	if err != nil {
		setErrorCondition(r.dk.Conditions(), err.Error())

		return err
	}

	setExistsCondition(r.dk.Conditions())
	log.Info("log monitoring setting created", "settings", objectID)

	return nil
}

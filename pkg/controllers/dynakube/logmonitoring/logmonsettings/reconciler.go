package logmonsettings

import (
	"context"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/logmonitoring"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/meta"
)

type reconciler struct {
	dk  *dynakube.DynaKube
	dtc dtclient.Client

	timeProvider *timeprovider.Provider
}

type ReconcilerBuilder func(dtc dtclient.Client, dk *dynakube.DynaKube) controllers.Reconciler

var _ ReconcilerBuilder = NewReconciler

func NewReconciler(dtc dtclient.Client, dk *dynakube.DynaKube) controllers.Reconciler {
	return &reconciler{
		dk:           dk,
		dtc:          dtc,
		timeProvider: timeprovider.New(),
	}
}

func (r *reconciler) Reconcile(ctx context.Context) error {
	const apiName = "log-monitoring-settings"
	apiProps := map[string]string{"meid": r.dk.Status.KubernetesClusterMEID}

	if !r.dk.Status.DynatraceAPI.IsRequestAllowed(apiName, apiProps) {
		return nil
	}

	if !r.dk.LogMonitoring().IsEnabled() {
		_ = meta.RemoveStatusCondition(r.dk.Conditions(), ConditionType)

		return nil
	}

	hasReadScope := conditions.IsOptionalScopeAvailable(r.dk, dtclient.ConditionTypeAPITokenSettingsRead)
	hasWriteScope := conditions.IsOptionalScopeAvailable(r.dk, dtclient.ConditionTypeAPITokenSettingsWrite)

	var missingScopes []string
	if !hasReadScope {
		missingScopes = append(missingScopes, dtclient.TokenScopeSettingsRead)
	}

	if !hasWriteScope {
		missingScopes = append(missingScopes, dtclient.TokenScopeSettingsWrite)
	}

	if len(missingScopes) > 0 {
		message := strings.Join(missingScopes, ", ") + " scope(s) missing: cannot query existing log monitoring setting and/or safely create new one."
		conditions.SetOptionalScopeMissing(r.dk.Conditions(), ConditionType, message)
		log.Info(message)

		return nil
	} else {
		log.Info("necessary scopes for logmonitoring settings creation is available, proceeding with reconciliation")
	}

	err := r.checkLogMonitoringSettings(ctx)
	if err != nil {
		return err
	}

	r.dk.Status.DynatraceAPI.AddRequest(apiName, apiProps)

	return nil
}

func (r *reconciler) checkLogMonitoringSettings(ctx context.Context) error {
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

		setAlreadyExistsCondition(r.dk.Conditions())

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

	setCreatedCondition(r.dk.Conditions())
	log.Info("log monitoring setting created", "settings", objectID)

	return nil
}

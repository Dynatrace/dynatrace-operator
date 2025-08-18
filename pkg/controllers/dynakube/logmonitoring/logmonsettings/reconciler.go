package logmonsettings

import (
	"context"
	"slices"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/logmonitoring"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/logmonitoring/daemonset"
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
	if !conditions.IsOutdated(r.timeProvider, r.dk, ConditionType) {
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
		missingScopes = append(missingScopes, "settings.read")
	}

	if !hasWriteScope {
		missingScopes = append(missingScopes, "settings.write")
	}

	missingScopeThreshold := 2
	if len(missingScopes) == missingScopeThreshold {
		message := strings.Join(missingScopes, ", ") + " scope(s) missing: cannot query/create or update existing settings"
		setLogMonitoringSettingError(r.dk.Conditions(), ConditionType, message)
		log.Info("LogMonitoring settings are not available due to missing scopes, will skip reconciliation")

		return nil
	} else {
		log.Info("LogMonitoring settings are available, proceeding with reconciliation")

		_ = meta.RemoveStatusCondition(r.dk.Conditions(), ConditionType)
	}

	if r.dk.Status.KubernetesClusterMEID == "" {
		log.Info("LogMonitoring settings are not yet available, which are needed to use logmonitoring settings, will requeue")

		return daemonset.KubernetesSettingsNotAvailableError
	}

	err := r.checkLogMonitoringSettings(ctx, missingScopes)
	if err != nil {
		return err
	}

	return nil
}

func (r *reconciler) checkLogMonitoringSettings(ctx context.Context, missingScopes []string) error {
	log.Info("start reconciling log monitoring settings")

	var (
		err                   error
		logMonitoringSettings dtclient.GetLogMonSettingsResponse
	)

	if !slices.Contains(missingScopes, "settings.read") {
		logMonitoringSettings, err = r.dtc.GetSettingsForLogModule(ctx, r.dk.Status.KubernetesClusterMEID)
		if err != nil {
			setLogMonitoringSettingError(r.dk.Conditions(), ConditionType, err.Error())

			return errors.WithMessage(err, "error trying to check if setting exists")
		}
	}

	if logMonitoringSettings.TotalCount > 0 {
		log.Info("there are already settings", "settings", logMonitoringSettings)

		setLogMonitoringSettingExists(r.dk.Conditions(), ConditionType)

		return nil
	}

	matchers := []logmonitoring.IngestRuleMatchers{}
	if len(r.dk.LogMonitoring().IngestRuleMatchers) > 0 {
		matchers = r.dk.LogMonitoring().IngestRuleMatchers
	}

	if !slices.Contains(missingScopes, "settings.write") {
		objectID, err := r.dtc.CreateLogMonitoringSetting(ctx, r.dk.Status.KubernetesClusterMEID, r.dk.Status.KubernetesClusterName, matchers)
		if err != nil {
			setLogMonitoringSettingError(r.dk.Conditions(), ConditionType, err.Error())

			return err
		}

		setLogMonitoringSettingCreated(r.dk.Conditions(), ConditionType)
		log.Info("log monitoring setting created", "settings", objectID)
	}

	return nil
}

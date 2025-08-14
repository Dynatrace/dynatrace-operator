package logmonsettings

import (
	"context"

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
		meta.RemoveStatusCondition(r.dk.Conditions(), ConditionType)

		return nil
	}

	hasReadScope := conditions.IsOptionalScopeAvailable(r.dk, dtclient.ConditionTypeAPITokenSettingsRead)
	hasWriteScope := conditions.IsOptionalScopeAvailable(r.dk, dtclient.ConditionTypeAPITokenSettingsWrite)

	if !hasReadScope {
		setLogMonitoringSettingError(r.dk.Conditions(), ConditionType, "settings.read scope missing: cannot query existing settings")
	}

	if !hasWriteScope {
		setLogMonitoringSettingError(r.dk.Conditions(), ConditionType, "settings.write scope missing: cannot create or update settings")
	}

	if !hasReadScope && !hasWriteScope {
		log.Info("LogMonitoring settings are not available due to missing scopes, will skip reconciliation")

		return nil
	} else if !hasReadScope && r.dk.Status.KubernetesClusterMEID == "" {
		log.Info("LogMonitoring settings are not yet available and settings.read scope is missing, will skip reconciliation")

		return nil
	}

	if r.dk.Status.KubernetesClusterMEID == "" {
		log.Info("LogMonitoring settings are not yet available, which are needed to use logmonitoring settings, will requeue")

		return daemonset.KubernetesSettingsNotAvailableError
	}

	err := r.checkLogMonitoringSettings(ctx, hasReadScope, hasWriteScope)
	if err != nil {
		return err
	}

	return nil
}

func (r *reconciler) checkLogMonitoringSettings(ctx context.Context, hasReadScope, hasWriteScope bool) error {
	log.Info("start reconciling log monitoring settings")

	var (
		err                   error
		logMonitoringSettings dtclient.GetLogMonSettingsResponse
	)

	if hasReadScope {
		logMonitoringSettings, err = r.dtc.GetSettingsForLogModule(ctx, r.dk.Status.KubernetesClusterMEID)
		if err != nil {
			setLogMonitoringSettingError(r.dk.Conditions(), ConditionType, err.Error())

			return errors.WithMessage(err, "error trying to check if setting exists")
		}

		if logMonitoringSettings.TotalCount > 0 {
			log.Info("there are already settings", "settings", logMonitoringSettings)

			setLogMonitoringSettingExists(r.dk.Conditions(), ConditionType)

			return nil
		}
	}

	if logMonitoringSettings.TotalCount > 0 {
		log.Info("there are already settings", "settings", logMonitoringSettings)

		setLogMonitoringSettingExists(r.dk.Conditions(), ConditionType)

		return nil
	}

	if hasWriteScope {
		matchers := []logmonitoring.IngestRuleMatchers{}
		if len(r.dk.LogMonitoring().IngestRuleMatchers) > 0 {
			matchers = r.dk.LogMonitoring().IngestRuleMatchers
		}

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

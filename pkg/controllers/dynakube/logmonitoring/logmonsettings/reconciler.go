package logmonsettings

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube/logmonitoring"
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
	} else if r.dk.Status.KubernetesClusterMEID == "" {
		log.Info("Kubernetes settings are not yet available, which are needed for LogMonitoring, will requeue")

		return daemonset.KubernetesSettingsNotAvailableError
	}

	err := r.checkLogMonitoringSettings(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (r *reconciler) checkLogMonitoringSettings(ctx context.Context) error {
	log.Info("start reconciling log monitoring settings")

	logMonitoringSettings, err := r.dtc.GetSettingsForLogModule(ctx, r.dk.Status.KubernetesClusterMEID)
	if err != nil {
		setLogMonitoringSettingError(r.dk.Conditions(), ConditionType, err.Error())

		return errors.WithMessage(err, "error trying to check if setting exists")
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

	objectId, err := r.dtc.CreateLogMonitoringSetting(ctx, r.dk.Status.KubernetesClusterMEID, r.dk.Status.KubernetesClusterName, matchers)

	if err != nil {
		setLogMonitoringSettingError(r.dk.Conditions(), ConditionType, err.Error())

		return err
	}

	log.Info("logmonitoring setting created", "settings", objectId)

	return nil
}

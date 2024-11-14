package logmonitoring

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/logmonitoring"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	oaconnectioninfo "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/logmonitoring/configsecret"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/logmonitoring/daemonset"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/monitoredentities"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler struct {
	client    client.Client
	apiReader client.Reader
	dk        *dynakube.DynaKube
	dtc       dtclient.Client

	configSecretReconciler           controllers.Reconciler
	daemonsetReconciler              controllers.Reconciler
	oneAgentConnectionInfoReconciler controllers.Reconciler
	monitoredEntitiesReconciler      controllers.Reconciler
}

type ReconcilerBuilder func(clt client.Client, apiReader client.Reader, dtc dtclient.Client, dk *dynakube.DynaKube) controllers.Reconciler

func NewReconciler(clt client.Client,
	apiReader client.Reader,
	dtc dtclient.Client,
	dk *dynakube.DynaKube) controllers.Reconciler {
	return &Reconciler{
		client:    clt,
		apiReader: apiReader,
		dk:        dk,
		dtc:       dtc,

		configSecretReconciler:           configsecret.NewReconciler(clt, apiReader, dk),
		daemonsetReconciler:              daemonset.NewReconciler(clt, apiReader, dk),
		oneAgentConnectionInfoReconciler: oaconnectioninfo.NewReconciler(clt, apiReader, dtc, dk),
		monitoredEntitiesReconciler:      monitoredentities.NewReconciler(dtc, dk),
	}
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	err := r.monitoredEntitiesReconciler.Reconcile(ctx)
	if err != nil {
		return err
	}

	err = r.oneAgentConnectionInfoReconciler.Reconcile(ctx)
	if err != nil {
		return err
	}

	err = r.configSecretReconciler.Reconcile(ctx)
	if err != nil {
		return err
	}

	err = r.daemonsetReconciler.Reconcile(ctx)
	if err != nil {
		return err
	}

	err = r.checkLogMonitoringSettings(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) checkLogMonitoringSettings(ctx context.Context) error {
	logMonitoringSettings, err := r.dtc.GetSettingsForLogModule(ctx, r.dk.Status.KubernetesClusterMEID)
	if err != nil {
		return errors.WithMessage(err, "error trying to check if setting exists")
	}

	if logMonitoringSettings.TotalCount > 0 {
		log.Info("there are already settings", "settings", logMonitoringSettings)

		conditions.SetLogMonitoringSettingExists(r.dk.Conditions(), conditionType)

		return nil
	} else if logMonitoringSettings.TotalCount == 0 {
		matchers := []logmonitoring.IngestRuleMatchers{}
		if len(r.dk.LogMonitoring().IngestRuleMatchers) > 0 {
			matchers = r.dk.LogMonitoring().IngestRuleMatchers
		}

		objectId, err := r.dtc.CreateLogMonitoringSetting(ctx, r.dk.Status.KubernetesClusterMEID, r.dk.Status.KubernetesClusterName, matchers)

		if err != nil {
			return errors.WithMessage(err, "error when creating log monitoring setting")
		}

		log.Info("logmonitoring setting created", "settings", objectId)
	}

	return nil
}

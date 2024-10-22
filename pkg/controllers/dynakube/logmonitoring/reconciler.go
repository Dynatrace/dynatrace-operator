package logmonitoring

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	oaconnectioninfo "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/logmonitoring/configsecret"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/logmonitoring/daemonset"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/monitoredentities"
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

	return nil
}

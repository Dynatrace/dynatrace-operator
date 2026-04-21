package logmonitoring

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/settings"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	oaconnectioninfo "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/logmonitoring/configsecret"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/logmonitoring/daemonset"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/logmonitoring/logmonsettings"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type subReconciler interface {
	Reconcile(ctx context.Context, dk *dynakube.DynaKube) error
}

type logmonsettingsSubReconciler interface {
	Reconcile(ctx context.Context, dtClient settings.APIClient, dk *dynakube.DynaKube) error
}

type Reconciler struct {
	client    client.Client
	apiReader client.Reader

	configSecretReconciler           subReconciler
	daemonsetReconciler              subReconciler
	oneAgentConnectionInfoReconciler controllers.Reconciler
	logmonsettingsReconciler         logmonsettingsSubReconciler
}

func NewReconciler(clt client.Client, apiReader client.Reader) *Reconciler {
	return &Reconciler{
		client:    clt,
		apiReader: apiReader,

		configSecretReconciler:   configsecret.NewReconciler(clt, apiReader),
		daemonsetReconciler:      daemonset.NewReconciler(clt, apiReader),
		logmonsettingsReconciler: logmonsettings.NewReconciler(),
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, dtClient *dynatrace.Client, dk *dynakube.DynaKube) error {
	oaConnectionInfoReconciler := r.oneAgentConnectionInfoReconciler
	if oaConnectionInfoReconciler == nil {
		oaConnectionInfoReconciler = oaconnectioninfo.NewReconciler(r.client, r.apiReader, dtClient.OneAgent, dk)
	}

	err := oaConnectionInfoReconciler.Reconcile(ctx)
	if err != nil {
		return err
	}

	err = r.configSecretReconciler.Reconcile(ctx, dk)
	if err != nil {
		return err
	}

	err = r.daemonsetReconciler.Reconcile(ctx, dk)
	if err != nil {
		return err
	}

	err = r.logmonsettingsReconciler.Reconcile(ctx, dtClient.Settings, dk)
	if err != nil {
		return err
	}

	return nil
}

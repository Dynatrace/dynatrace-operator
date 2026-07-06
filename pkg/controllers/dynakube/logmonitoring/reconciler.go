package logmonitoring

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	dtimage "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/image"
	oaClient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/settings"
	oaconnectioninfo "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/logmonitoring/configsecret"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/logmonitoring/daemonset"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/logmonitoring/logmonsettings"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type subReconciler interface {
	Reconcile(ctx context.Context, dk *dynakube.DynaKube) error
}

type imageAwareSubReconciler interface {
	Reconcile(ctx context.Context, imageClient dtimage.Client, dk *dynakube.DynaKube) error
}

type logmonsettingsSubReconciler interface {
	Reconcile(ctx context.Context, dtClient settings.Client, dk *dynakube.DynaKube) error
}

type oaConnectionInfoReconciler interface {
	Reconcile(ctx context.Context, oaClient oaClient.Client, dk *dynakube.DynaKube) error
}

type Reconciler struct {
	client    client.Client
	apiReader client.Reader

	configSecretReconciler           subReconciler
	daemonsetReconciler              imageAwareSubReconciler
	oneAgentConnectionInfoReconciler oaConnectionInfoReconciler
	logmonsettingsReconciler         logmonsettingsSubReconciler
}

func NewReconciler(clt client.Client, apiReader client.Reader) *Reconciler {
	return &Reconciler{
		client:    clt,
		apiReader: apiReader,

		configSecretReconciler:           configsecret.NewReconciler(clt, apiReader),
		daemonsetReconciler:              daemonset.NewReconciler(clt, apiReader),
		logmonsettingsReconciler:         logmonsettings.NewReconciler(),
		oneAgentConnectionInfoReconciler: oaconnectioninfo.NewReconciler(clt, apiReader),
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, dtClient *dynatrace.Client, dk *dynakube.DynaKube) error {
	ctx, _ = logd.NewFromContext(ctx, "logmonitoring")

	err := r.oneAgentConnectionInfoReconciler.Reconcile(ctx, dtClient.OneAgent, dk)
	if err != nil {
		return err
	}

	err = r.configSecretReconciler.Reconcile(ctx, dk)
	if err != nil {
		return err
	}

	err = r.daemonsetReconciler.Reconcile(ctx, dtClient.Images, dk)
	if err != nil {
		return err
	}

	err = r.logmonsettingsReconciler.Reconcile(ctx, dtClient.Settings, dk)
	if err != nil {
		return err
	}

	return nil
}

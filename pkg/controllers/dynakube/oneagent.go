package dynakube

import (
	"context"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/version"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/object"
	"k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (controller *Controller) reconcileOneAgent(ctx context.Context, dynakube *dynakube.DynaKube, versionReconciler *version.Reconciler) error {
	if !dynakube.NeedsOneAgent() {
		return controller.removeOneAgentDaemonSet(ctx, dynakube)
	}
	err := versionReconciler.ReconcileOA(ctx)
	if err != nil {
		return err
	}

	return oneagent.NewOneAgentReconciler(
		controller.client, controller.apiReader, controller.scheme, controller.clusterID,
	).Reconcile(ctx, dynakube)
}

func (controller *Controller) removeOneAgentDaemonSet(ctx context.Context, dynakube *dynakube.DynaKube) error {
	oneAgentDaemonSet := v1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: dynakube.OneAgentDaemonsetName(), Namespace: dynakube.Namespace}}
	return object.Delete(ctx, controller.client, &oneAgentDaemonSet)
}

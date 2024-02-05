package dynakube

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/version"
)

func (controller *Controller) reconcileOneAgent(ctx context.Context, dynakube *dynakube.DynaKube, versionReconciler version.Reconciler) error {
	if dynakube.NeedsOneAgent() {
		err := versionReconciler.ReconcileOneAgent(ctx, dynakube)
		if err != nil {
			return err
		}
	}
	return oneagent.NewOneAgentReconciler(
		controller.client, controller.apiReader, controller.scheme, controller.clusterID,
	).Reconcile(ctx, dynakube)
}

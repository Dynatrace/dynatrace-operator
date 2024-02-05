package dynakube

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/apimonitoring"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/istio"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/version"
	"github.com/pkg/errors"
)

func (controller *Controller) reconcileActiveGate(ctx context.Context, dynakube *dynakube.DynaKube, dtc dynatrace.Client, istioReconciler istio.Reconciler, connectionReconciler connectioninfo.Reconciler, versionReconciler version.Reconciler) error { // nolint: revive
	if dynakube.NeedsActiveGate() { // TODO: this is not optimal, because this check is in the activegate reconciler as well (to do the cleanup)
		err := connectionReconciler.ReconcileActiveGate(ctx, dynakube)
		if err != nil {
			return err
		}

		err = versionReconciler.ReconcileActiveGate(ctx, dynakube)
		if err != nil {
			return err
		}

		if istioReconciler != nil {
			err = istioReconciler.ReconcileActiveGateCommunicationHosts(ctx, dynakube)
			if err != nil {
				return err
			}
		}
	} // TODO: have a cleanup for things that we create above

	reconciler := controller.activeGateReconcilerBuilder(controller.client, controller.apiReader, controller.scheme, dynakube, dtc)
	err := reconciler.Reconcile(ctx)

	if err != nil {
		return errors.WithMessage(err, "failed to reconcile ActiveGate")
	}

	controller.setupAutomaticApiMonitoring(dynakube, dtc)

	return nil
}

func (controller *Controller) setupAutomaticApiMonitoring(dynakube *dynakube.DynaKube, dtc dynatrace.Client) {
	if dynakube.Status.KubeSystemUUID != "" &&
		dynakube.FeatureAutomaticKubernetesApiMonitoring() &&
		dynakube.IsKubernetesMonitoringActiveGateEnabled() {
		clusterLabel := dynakube.FeatureAutomaticKubernetesApiMonitoringClusterName()
		if clusterLabel == "" {
			clusterLabel = dynakube.Name
		}

		err := apimonitoring.NewReconciler(dtc, clusterLabel, dynakube.Status.KubeSystemUUID).
			Reconcile(dynakube)
		if err != nil {
			log.Error(err, "could not create setting")
		}
	}
}

package dynakube

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/istio"
	"github.com/pkg/errors"
)

func (controller *Controller) reconcileActiveGate(ctx context.Context, dynakube *dynakube.DynaKube, dtc dynatrace.Client, istioClient *istio.Client) error {
	reconciler := controller.activeGateReconcilerBuilder(controller.client, controller.apiReader, controller.scheme, dynakube, dtc, istioClient)
	err := reconciler.Reconcile(ctx)

	if err != nil {
		return errors.WithMessage(err, "failed to reconcile ActiveGate")
	}

	controller.setupAutomaticApiMonitoring(ctx, dtc, dynakube)

	return nil
}

func (controller *Controller) setupAutomaticApiMonitoring(ctx context.Context, dtc dynatrace.Client, dynakube *dynakube.DynaKube) {
	if dynakube.Status.KubeSystemUUID != "" &&
		dynakube.FeatureAutomaticKubernetesApiMonitoring() &&
		dynakube.IsKubernetesMonitoringActiveGateEnabled() {
		clusterLabel := dynakube.FeatureAutomaticKubernetesApiMonitoringClusterName()
		if clusterLabel == "" {
			clusterLabel = dynakube.Name
		}

		err := controller.apiMonitoringReconcilerBuilder(dtc, dynakube, clusterLabel, dynakube.Status.KubeSystemUUID).
			Reconcile(ctx)
		if err != nil {
			log.Error(err, "could not create setting")
		}
	}
}

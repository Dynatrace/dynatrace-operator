package dynakube

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/istio"
	"github.com/pkg/errors"
)

func (controller *Controller) reconcileActiveGate(ctx context.Context, dk *dynakube.DynaKube, dtc dynatrace.Client, istioClient *istio.Client) error {
	reconciler := controller.activeGateReconcilerBuilder(controller.client, controller.apiReader, dk, dtc, istioClient, controller.tokens)

	err := reconciler.Reconcile(ctx)
	if err != nil {
		return errors.WithMessage(err, "failed to reconcile ActiveGate")
	}

	controller.setupAutomaticAPIMonitoring(ctx, dtc, dk)

	return nil
}

func (controller *Controller) setupAutomaticAPIMonitoring(ctx context.Context, dtc dynatrace.Client, dk *dynakube.DynaKube) {
	if dk.Status.KubeSystemUUID != "" &&
		dk.FF().IsAutomaticK8sAPIMonitoring() &&
		dk.ActiveGate().IsKubernetesMonitoringEnabled() {
		clusterLabel := dk.FF().GetAutomaticK8sAPIMonitoringClusterName()
		if clusterLabel == "" {
			clusterLabel = dk.Name
		}

		err := controller.apiMonitoringReconcilerBuilder(dtc.AsV2().Settings, dk, clusterLabel).
			Reconcile(ctx)
		if err != nil {
			log.Error(err, "could not create setting")
		}
	}
}

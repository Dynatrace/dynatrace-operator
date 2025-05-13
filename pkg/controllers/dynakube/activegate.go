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

	controller.setupAutomaticApiMonitoring(ctx, dtc, dk)

	return nil
}

func (controller *Controller) setupAutomaticApiMonitoring(ctx context.Context, dtc dynatrace.Client, dk *dynakube.DynaKube) {
	if dk.Status.KubeSystemUUID != "" &&
		dk.FF().IsAutomaticInjection() &&
		dk.ActiveGate().IsKubernetesMonitoringEnabled() {
		clusterLabel := dk.FF().GetAutomaticK8sApiMonitoringClusterName()
		if clusterLabel == "" {
			clusterLabel = dk.Name
		}

		err := controller.apiMonitoringReconcilerBuilder(dtc, dk, clusterLabel).
			Reconcile(ctx)
		if err != nil {
			log.Error(err, "could not create setting")
		}
	}
}

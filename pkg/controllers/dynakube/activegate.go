package dynakube

import (
	"context"
	"errors"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
)

func (controller *Controller) reconcileActiveGate(ctx context.Context, dk *dynakube.DynaKube, dtc dynatrace.Client) error {
	var errs []error

	// this is a temporary fix, setupAutomaticAPIMonitoring will try to create/update the k8s connection setting if we do not beforehand check for the ME for this cluster.
	if err := controller.k8sEntityReconciler.Reconcile(ctx, dtc.AsV2().Settings, dk); err != nil {
		errs = append(errs, err)
	}

	reconciler := controller.activeGateReconcilerBuilder(controller.client, controller.apiReader, dk, dtc, controller.tokens)

	err := reconciler.Reconcile(ctx)
	if err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
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

		settingsClient := dtc.AsV2().Settings

		err := controller.apiMonitoringReconciler.Reconcile(ctx, settingsClient, clusterLabel, dk)
		if err != nil {
			log.Error(err, "could not create setting")

			return
		}

		// this is a temporary fix, apiMonitoringReconciler will only create the k8s connection setting, but will not set the ME related info that the setting creation causes.
		if err := controller.k8sEntityReconciler.Reconcile(ctx, settingsClient, dk); err != nil {
			log.Error(err, "could not reconcile k8s entity after setting creation")
		}
	}
}

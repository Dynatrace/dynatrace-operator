package monitoredentities

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
)

type Reconciler struct {
	dk           *dynakube.DynaKube
	dtClient     dynatrace.Client
	timeProvider *timeprovider.Provider
}

type ReconcilerBuilder func(dtClient dynatrace.Client, dk *dynakube.DynaKube) controllers.Reconciler

func NewReconciler( //nolint
	dtClient dynatrace.Client,
	dk *dynakube.DynaKube,
) controllers.Reconciler {
	return &Reconciler{
		dk:           dk,
		dtClient:     dtClient,
		timeProvider: timeprovider.New(),
	}
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	log.Info("start reconciling monitored entities")

	if !conditions.IsOutdated(r.timeProvider, r.dk, MEIDConditionType) {
		log.Info("Kubernetes Cluster MEID not outdated, skipping reconciliation")

		return nil
	}

	conditions.SetStatusOutdated(r.dk.Conditions(), MEIDConditionType, "Kubernetes Cluster MEID is outdated in the status")

	if !conditions.IsOptionalScopeAvailable(r.dk, dynatrace.ConditionTypeAPITokenSettingsRead) {
		log.Info(dynatrace.TokenScopeSettingsRead + " optional scope not available")

		return nil
	}

	k8sEntity, err := r.dtClient.GetKubernetesClusterEntity(ctx, r.dk.Status.KubeSystemUUID)
	if err != nil {
		log.Info("failed to retrieve MEs")

		return err
	}

	if k8sEntity.ID == "" {
		log.Info("no MEs found, no kubernetesClusterMEID will be set in the dynakube status")

		return nil
	}

	r.dk.Status.KubernetesClusterMEID = k8sEntity.ID
	r.dk.Status.KubernetesClusterName = k8sEntity.Name
	conditions.SetStatusUpdated(r.dk.Conditions(), MEIDConditionType, "Kubernetes Cluster MEID is up to date")

	log.Info("kubernetesClusterMEID set in dynakube status, done reconciling", "kubernetesClusterMEID", r.dk.Status.KubernetesClusterMEID)

	return nil
}

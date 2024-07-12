package monitoredentities

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
)

type ReconcilerBuilder func(
	dtClient dynatrace.Client,
	dk *dynakube.DynaKube,
	kubeSystemUUID string,
) controllers.Reconciler

func NewReconciler( //nolint
	dtClient dynatrace.Client,
	dk *dynakube.DynaKube,
	kubeSystemUUID string,
) controllers.Reconciler {
	return &Reconciler{
		dk:             dk,
		dtClient:       dtClient,
		kubeSystemUUID: kubeSystemUUID,
	}
}

type Reconciler struct {
	dk             *dynakube.DynaKube
	dtClient       dynatrace.Client
	kubeSystemUUID string
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	if r.dk.Status.KubernetesClusterMEID == "" {
		monitoredEntities, err := r.dtClient.GetMonitoredEntitiesForKubeSystemUUID(ctx, r.kubeSystemUUID)
		if err != nil {
			log.Info("failed to retrieve MEs")

			return err
		}

		log.Info("retrieved MEs")

		if len(monitoredEntities) == 0 {
			log.Info("no MEs found, no monitoredentityID will be set in the dynakube status")

			return nil
		}

		r.dk.Status.KubernetesClusterMEID = findLatestEntity(monitoredEntities).EntityId
	}

	log.Info("kubernetesClusterMEID set in dynakube status, done reconciling", "KubernetesClusterMEID", r.dk.Status.KubernetesClusterMEID)

	return nil
}

func findLatestEntity(monitoredEntities []dynatrace.MonitoredEntity) dynatrace.MonitoredEntity {
	latest := monitoredEntities[0]
	for _, entity := range monitoredEntities {
		if entity.LastSeenTms < latest.LastSeenTms {
			latest = entity
		}
	}

	return latest
}

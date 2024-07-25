package monitoredentities

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
)

type ReconcilerBuilder func(
	dtClient dynatrace.Client,
	dk *dynakube.DynaKube,
) controllers.Reconciler

func NewReconciler( //nolint
	dtClient dynatrace.Client,
	dk *dynakube.DynaKube,
) controllers.Reconciler {
	return &Reconciler{
		dk:       dk,
		dtClient: dtClient,
	}
}

type Reconciler struct {
	dk       *dynakube.DynaKube
	dtClient dynatrace.Client
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	if r.dk.Status.KubernetesClusterMEID == "" {
		log.Info("start reconciling monitored entities")

		monitoredEntities, err := r.dtClient.GetMonitoredEntitiesForKubeSystemUUID(ctx, r.dk.Status.KubeSystemUUID)
		if err != nil {
			log.Info("failed to retrieve MEs")

			return err
		}

		if len(monitoredEntities) == 0 {
			log.Info("no MEs found, no kubernetesClusterMEID will be set in the dynakube status")

			return nil
		}

		r.dk.Status.KubernetesClusterMEID = findLatestEntity(monitoredEntities).EntityId

		log.Info("kubernetesClusterMEID set in dynakube status, done reconciling", "kubernetesClusterMEID", r.dk.Status.KubernetesClusterMEID)
	}

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

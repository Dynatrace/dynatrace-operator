package monitoredentities

import (
	"context"
	"errors"

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
	kubeSystemUUID string,
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
	log.Info("start reconciling monitored entities")

	if r.dk.Status.KubernetesClusterMEID == "" {
		monitoredEntities, err := r.dtClient.GetMonitoredEntitiesForKubeSystemUUID(ctx, r.dk.Status.KubeSystemUUID)
		if err != nil {
			log.Info("failed to retrieve MEs")

			return err
		}

		log.Info("retrieved MEs")

		if len(monitoredEntities) == 0 {
			return errors.New("MEs are empty, at this point this should not be the case")
		}

		r.dk.Status.KubernetesClusterMEID = findLatestEntity(monitoredEntities).EntityId
	}

	log.Info("kubernetesClusterMEID set in dynakube status, done reconciling", "kubernetesClusterMEID", r.dk.Status.KubernetesClusterMEID)

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

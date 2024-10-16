package apimonitoring

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/monitoredentities"
	"github.com/pkg/errors"
)

type Reconciler struct {
	dtc                         dtclient.Client
	dk                          *dynakube.DynaKube
	monitoredEntitiesReconciler monitoredentities.ReconcilerBuilder
	clusterLabel                string
}

type ReconcilerBuilder func(dtc dtclient.Client, dk *dynakube.DynaKube, clusterLabel string) *Reconciler

func NewReconciler(dtc dtclient.Client, dk *dynakube.DynaKube, clusterLabel string) *Reconciler {
	return &Reconciler{
		dtc:                         dtc,
		dk:                          dk,
		clusterLabel:                clusterLabel,
		monitoredEntitiesReconciler: monitoredentities.NewReconciler,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	objectID, err := r.createObjectIdIfNotExists(ctx)
	if err != nil {
		return err
	}

	if objectID != "" {
		log.Info("created kubernetes cluster setting", "clusterLabel", r.clusterLabel, "cluster", r.dk.Status.KubeSystemUUID, "object id", objectID)
	} else {
		log.Info("kubernetes cluster setting already exists", "clusterLabel", r.clusterLabel, "cluster", r.dk.Status.KubeSystemUUID)
	}

	return nil
}

func (r *Reconciler) createObjectIdIfNotExists(ctx context.Context) (string, error) {
	if r.dk.Status.KubeSystemUUID == "" {
		return "", errors.New("no kube-system namespace UUID given")
	}

	err := r.monitoredEntitiesReconciler(r.dtc, r.dk).Reconcile(ctx)
	if err != nil {
		return "", err
	}

	var monitoredEntity []dtclient.MonitoredEntity

	if r.dk.Status.KubernetesClusterMEID != "" {
		monitoredEntity = []dtclient.MonitoredEntity{
			{
				EntityId: r.dk.Status.KubernetesClusterMEID,
			},
		}
	}

	// check if Setting for ME exists
	settings, err := r.dtc.GetSettingsForMonitoredEntities(ctx, monitoredEntity, dtclient.KubernetesSettingsSchemaId)
	if err != nil {
		return "", errors.WithMessage(err, "error trying to check if setting exists")
	}

	if settings.TotalCount > 0 {
		_, err = r.handleKubernetesAppEnabled(ctx, monitoredEntity)
		if err != nil {
			return "", err
		}

		return "", nil
	}

	objectID, err := r.dtc.CreateOrUpdateKubernetesSetting(ctx, r.clusterLabel, r.dk.Status.KubeSystemUUID, r.dk.Status.KubernetesClusterMEID)
	if err != nil {
		return "", errors.WithMessage(err, "error creating dynatrace settings object")
	}

	if r.dk.Status.KubernetesClusterMEID == "" {
		// the CreateOrUpdateKubernetesSetting call will create the ME(monitored-entity) if no scope was given (scope == entity-id), this happens on the "first run"
		// so we have to run the entity reconciler AGAIN to set it in the status.
		err := r.monitoredEntitiesReconciler(r.dtc, r.dk).Reconcile(ctx)
		if err != nil {
			return "", err
		}
	}

	return objectID, nil
}

func (r *Reconciler) handleKubernetesAppEnabled(ctx context.Context, monitoredEntities []dtclient.MonitoredEntity) (string, error) {
	if r.dk.FeatureEnableK8sAppEnabled() {
		appSettings, err := r.dtc.GetSettingsForMonitoredEntities(ctx, monitoredEntities, dtclient.AppTransitionSchemaId)
		if err != nil {
			return "", errors.WithMessage(err, "error trying to check if app setting exists")
		}

		if appSettings.TotalCount == 0 {
			meID := determineNewestMonitoredEntity(monitoredEntities)
			if meID != "" {
				transitionSchemaObjectID, err := r.dtc.CreateOrUpdateKubernetesAppSetting(ctx, meID)
				if err != nil {
					log.Info("schema app-transition.kubernetes failed to set", "meID", meID, "err", err)

					return "", err
				} else {
					log.Info("schema app-transition.kubernetes set to true", "meID", meID, "transitionSchemaObjectID", transitionSchemaObjectID)

					return transitionSchemaObjectID, nil
				}
			}
		}
	}

	return "", nil
}

// determineNewestMonitoredEntity returns the UUID of the newest entities; or empty string if the slice of entities is empty
func determineNewestMonitoredEntity(entities []dtclient.MonitoredEntity) string {
	if len(entities) == 0 {
		return ""
	}

	var newestMe dtclient.MonitoredEntity
	for _, entity := range entities {
		if entity.LastSeenTms > newestMe.LastSeenTms {
			newestMe = entity
		}
	}

	return newestMe.EntityId
}

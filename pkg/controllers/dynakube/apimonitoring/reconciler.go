package apimonitoring

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/pkg/errors"
)

type Reconciler struct {
	dtc            dtclient.Client
	dynakube       *dynakube.DynaKube
	clusterLabel   string
	kubeSystemUUID string
}

type ReconcilerBuilder func(dtc dtclient.Client, dk *dynakube.DynaKube, clusterLabel, kubeSystemUUID string) *Reconciler

func NewReconciler(dtc dtclient.Client, dk *dynakube.DynaKube, clusterLabel, kubeSystemUUID string) *Reconciler {
	return &Reconciler{
		dtc,
		dk,
		clusterLabel,
		kubeSystemUUID,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	objectID, err := r.createObjectIdIfNotExists(ctx)
	if err != nil {
		return err
	}

	if objectID != "" {
		log.Info("created kubernetes cluster setting", "clusterLabel", r.clusterLabel, "cluster", r.kubeSystemUUID, "object id", objectID)
	} else {
		log.Info("kubernetes cluster setting already exists", "clusterLabel", r.clusterLabel, "cluster", r.kubeSystemUUID)
	}

	return nil
}

func (r *Reconciler) createObjectIdIfNotExists(ctx context.Context) (string, error) {
	if r.kubeSystemUUID == "" {
		return "", errors.New("no kube-system namespace UUID given")
	}

	// check if ME with UID exists
	var monitoredEntities, err = r.dtc.GetMonitoredEntitiesForKubeSystemUUID(ctx, r.kubeSystemUUID)
	if err != nil {
		return "", errors.WithMessage(err, "error while loading MEs")
	}

	// check if Setting for ME exists
	settings, err := r.dtc.GetSettingsForMonitoredEntities(ctx, monitoredEntities, dtclient.KubernetesSettingsSchemaId)
	if err != nil {
		return "", errors.WithMessage(err, "error trying to check if setting exists")
	}

	if settings.TotalCount > 0 {
		_, err = r.handleKubernetesAppEnabled(ctx, monitoredEntities)
		if err != nil {
			return "", err
		}

		return "", nil
	}

	// determine newest ME (can be empty string), and create or update a settings object accordingly
	meID := determineNewestMonitoredEntity(monitoredEntities)

	objectID, err := r.dtc.CreateOrUpdateKubernetesSetting(ctx, r.clusterLabel, r.kubeSystemUUID, meID)
	if err != nil {
		return "", errors.WithMessage(err, "error creating dynatrace settings object")
	}

	return objectID, nil
}

func (r *Reconciler) handleKubernetesAppEnabled(ctx context.Context, monitoredEntities []dtclient.MonitoredEntity) (string, error) {
	if r.dynakube.FeatureEnableK8sAppEnabled() {
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

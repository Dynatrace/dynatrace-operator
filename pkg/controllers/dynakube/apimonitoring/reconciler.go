package apimonitoring

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube"
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

	var monitoredEntity *dtclient.MonitoredEntity

	if r.dk.Status.KubernetesClusterMEID != "" {
		monitoredEntity = &dtclient.MonitoredEntity{
			EntityId: r.dk.Status.KubernetesClusterMEID,
		}
	}

	// check if Setting for ME exists
	settings, err := r.dtc.GetSettingsForMonitoredEntity(ctx, monitoredEntity, dtclient.KubernetesSettingsSchemaId)
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

func (r *Reconciler) handleKubernetesAppEnabled(ctx context.Context, monitoredEntity *dtclient.MonitoredEntity) (string, error) {
	if r.dk.FeatureEnableK8sAppEnabled() {
		appSettings, err := r.dtc.GetSettingsForMonitoredEntity(ctx, monitoredEntity, dtclient.AppTransitionSchemaId)
		if err != nil {
			return "", errors.WithMessage(err, "error trying to check if app setting exists")
		}

		if appSettings.TotalCount == 0 {
			meID := monitoredEntity.EntityId
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

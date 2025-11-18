package apimonitoring

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/k8sentity"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/pkg/errors"
)

type Reconciler struct {
	dtc dtclient.Client

	k8sEntityReconciler controllers.Reconciler
	dk                  *dynakube.DynaKube
	clusterLabel        string
}

type ReconcilerBuilder func(dtc dtclient.Client, dk *dynakube.DynaKube, clusterLabel string) controllers.Reconciler

func NewReconciler(dtc dtclient.Client, dk *dynakube.DynaKube, clusterLabel string) controllers.Reconciler {
	return &Reconciler{
		dtc:                 dtc,
		dk:                  dk,
		clusterLabel:        clusterLabel,
		k8sEntityReconciler: k8sentity.NewReconciler(dtc, dk),
	}
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	if !conditions.IsOptionalScopeAvailable(r.dk, dtclient.ConditionTypeAPITokenSettingsRead) {
		log.Info("api token missing optional scope, skipping reconciliation", "scope", dtclient.TokenScopeSettingsRead)

		return nil
	}

	if !conditions.IsOptionalScopeAvailable(r.dk, dtclient.ConditionTypeAPITokenSettingsWrite) {
		log.Info("api token missing optional scope, skipping reconciliation", "scope", dtclient.TokenScopeSettingsWrite)

		return nil
	}

	objectID, err := r.createObjectIDIfNotExists(ctx)
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

func (r *Reconciler) createObjectIDIfNotExists(ctx context.Context) (string, error) {
	if r.dk.Status.KubeSystemUUID == "" {
		return "", errors.New("no kube-system namespace UUID given")
	}

	err := r.k8sEntityReconciler.Reconcile(ctx)
	if err != nil {
		return "", err
	}

	var k8sEntity dtclient.K8sClusterME

	if r.dk.Status.KubernetesClusterMEID != "" {
		k8sEntity = dtclient.K8sClusterME{
			ID: r.dk.Status.KubernetesClusterMEID,
		}
	}

	// check if Setting for ME exists
	settings, err := r.dtc.GetSettingsForMonitoredEntity(ctx, k8sEntity, dtclient.KubernetesSettingsSchemaID)
	if err != nil {
		return "", errors.WithMessage(err, "error trying to check if setting exists")
	}

	if settings.TotalCount > 0 {
		if err := r.handleKubernetesAppEnabled(ctx, k8sEntity); err != nil {
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
		err := r.k8sEntityReconciler.Reconcile(ctx)
		if err != nil {
			return "", err
		}
	}

	return objectID, nil
}

func (r *Reconciler) handleKubernetesAppEnabled(ctx context.Context, k8sEntity dtclient.K8sClusterME) error {
	if r.dk.FF().IsK8sAppEnabled() {
		appSettings, err := r.dtc.GetSettingsForMonitoredEntity(ctx, k8sEntity, dtclient.AppTransitionSchemaID)
		if err != nil {
			return errors.WithMessage(err, "error trying to check if app setting exists")
		}

		if appSettings.TotalCount == 0 {
			meID := k8sEntity.ID
			if meID != "" {
				transitionSchemaObjectID, err := r.dtc.CreateOrUpdateKubernetesAppSetting(ctx, meID)
				if err != nil {
					log.Info("schema app-transition.kubernetes failed to set", "meID", meID, "err", err)

					return err
				}

				log.Info("schema app-transition.kubernetes set to true", "meID", meID, "transitionSchemaObjectID", transitionSchemaObjectID)
			}
		}
	}

	return nil
}

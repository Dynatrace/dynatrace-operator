package apimonitoring

import (
	"context"
	goerrors "errors"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/settings"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	"github.com/pkg/errors"
)

var errMissingKubeSystemUUID = goerrors.New("no kube-system namespace UUID given")

type Reconciler struct{}

func NewReconciler() *Reconciler {
	return &Reconciler{}
}

func (r *Reconciler) Reconcile(ctx context.Context, dtc settings.APIClient, clusterLabel string, dk *dynakube.DynaKube) error {
	if !k8sconditions.IsOptionalScopeAvailable(dk, dtclient.ConditionTypeAPITokenSettingsRead) {
		log.Info("api token missing optional scope, skipping reconciliation", "scope", dtclient.TokenScopeSettingsRead)

		return nil
	}

	if !k8sconditions.IsOptionalScopeAvailable(dk, dtclient.ConditionTypeAPITokenSettingsWrite) {
		log.Info("api token missing optional scope, skipping reconciliation", "scope", dtclient.TokenScopeSettingsWrite)

		return nil
	}

	objectID, err := r.createObjectIDIfNotExists(ctx, dtc, clusterLabel, dk)
	if err != nil {
		return err
	}

	if objectID != "" {
		log.Info("created kubernetes cluster setting", "clusterLabel", clusterLabel, "cluster", dk.Status.KubeSystemUUID, "object id", objectID)
	} else {
		log.Info("kubernetes cluster setting already exists", "clusterLabel", clusterLabel, "cluster", dk.Status.KubeSystemUUID)
	}

	return nil
}

func (r *Reconciler) createObjectIDIfNotExists(ctx context.Context, dtc settings.APIClient, clusterLabel string, dk *dynakube.DynaKube) (string, error) {
	if dk.Status.KubeSystemUUID == "" {
		return "", errMissingKubeSystemUUID
	}

	var k8sEntity settings.K8sClusterME

	if dk.Status.KubernetesClusterMEID != "" {
		k8sEntity = settings.K8sClusterME{
			ID: dk.Status.KubernetesClusterMEID,
		}
	}

	// check if Setting for ME exists
	settings, err := dtc.GetSettingsForMonitoredEntity(ctx, k8sEntity, settings.KubernetesSettingsSchemaID)
	if err != nil {
		return "", errors.WithMessage(err, "error trying to check if setting exists")
	}

	if settings.TotalCount > 0 {
		if err := r.handleKubernetesAppEnabled(ctx, k8sEntity, dtc, dk); err != nil {
			return "", err
		}

		return "", nil
	}

	objectID, err := dtc.CreateOrUpdateKubernetesSetting(ctx, clusterLabel, dk.Status.KubeSystemUUID, dk.Status.KubernetesClusterMEID)
	if err != nil {
		return "", errors.WithMessage(err, "error creating dynatrace settings object")
	}

	return objectID, nil
}

func (r *Reconciler) handleKubernetesAppEnabled(ctx context.Context, k8sEntity settings.K8sClusterME, dtc settings.APIClient, dk *dynakube.DynaKube) error {
	if dk.FF().IsK8sAppEnabled() {
		appSettings, err := dtc.GetSettingsForMonitoredEntity(ctx, k8sEntity, settings.AppTransitionSchemaID)
		if err != nil {
			if !core.IsNotFound(err) {
				return errors.WithMessage(err, "error trying to check if app setting exists")
			}

			if shouldLogMissingAppTransitionSchema(k8sEntity.ID) {
				log.Info("skipping app-transition creation due to missing schema", "meID", k8sEntity.ID, "schemaID", settings.AppTransitionSchemaID)
			}

			return nil
		}

		if appSettings.TotalCount == 0 {
			meID := k8sEntity.ID
			if meID != "" {
				transitionSchemaObjectID, err := dtc.CreateOrUpdateKubernetesAppSetting(ctx, meID)
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

const logCacheTimeout = 5 * time.Minute

var logCache = make(map[string]time.Time)
var timeNow = time.Now

// NOT THREAD-SAFE!!!
func shouldLogMissingAppTransitionSchema(meID string) bool {
	// Limit cache size to prevent excessive memory usage at the cost of potentially spamming the logs.
	const maxCacheSize = 100
	if len(logCache) >= maxCacheSize {
		return true
	}

	lastLog, exists := logCache[meID]
	if !exists || timeNow().Sub(lastLog) > logCacheTimeout {
		logCache[meID] = timeNow()

		return true
	}

	return false
}

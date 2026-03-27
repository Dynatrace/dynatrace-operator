package k8sentity

import (
	"context"
	goerrors "errors"
	"fmt"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/settings"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/pkg/errors"
)

var errMissingKubeSystemUUID = goerrors.New("no kube-system namespace UUID given")

// meIDConditionType is the condition type used to cache the Kubernetes Cluster Monitored Entity ID in the DynaKube status.
const meIDConditionType = "MonitoredEntity"

type Reconciler struct {
	timeProvider *timeprovider.Provider
}

func NewReconciler() *Reconciler {
	return &Reconciler{
		timeProvider: timeprovider.New(),
	}
}

// Reconcile first looks up the Kubernetes Cluster Monitored Entity ID (MEID), then creates the
// builtin:cloud.kubernetes settings object if the DynaKube is configured for Kubernetes monitoring.
// On first run the MEID may not yet exist; after creating the settings object the MEID is immediately
// refreshed so that subsequent reconcilers do not need to wait for the next cycle.
func (r *Reconciler) Reconcile(ctx context.Context, dtc settings.APIClient, dk *dynakube.DynaKube) error {
	if !k8sconditions.IsOptionalScopeAvailable(dk, token.ConditionTypeAPITokenSettingsRead) {
		msg := token.ScopeSettingsRead + " optional scope not available"
		log.Info(msg)
		k8sconditions.SetOptionalScopeMissing(dk.Conditions(), meIDConditionType, msg)

		return nil
	}

	if dk.Status.KubeSystemUUID == "" {
		return errMissingKubeSystemUUID
	}

	if err := r.reconcileMEID(ctx, dtc, dk); err != nil {
		return err
	}

	if !dk.FF().IsAutomaticK8sAPIMonitoring() ||
		!dk.ActiveGate().IsKubernetesMonitoringEnabled() {
		return nil
	}

	if !k8sconditions.IsOptionalScopeAvailable(dk, token.ConditionTypeAPITokenSettingsWrite) {
		log.Info("api token missing optional scope, skipping reconciliation", "scope", token.ScopeSettingsWrite)

		return nil
	}

	objectID, err := r.createK8sConnectionSettingIfAbsent(ctx, dtc, dk)
	if err != nil {
		return err
	}

	if objectID != "" {
		// On first run the monitored entity only becomes available after the settings object is created.
		// Refresh the MEID immediately to avoid requiring an extra reconciliation cycle.
		if err := r.refreshClusterMEID(ctx, dtc, dk); err != nil {
			return err
		}
	}

	return r.createK8sAppSettingIfAbsent(ctx, dtc, dk)
}

// reconcileMEID fetches and caches the Kubernetes Cluster Monitored Entity ID in the DynaKube status.
// It uses time-based caching via the meIDConditionType condition; if the condition is still up to date,
// the API call is skipped.
func (r *Reconciler) reconcileMEID(ctx context.Context, dtc settings.APIClient, dk *dynakube.DynaKube) error {
	if !k8sconditions.IsOutdated(r.timeProvider, dk, meIDConditionType) {
		log.Info("Kubernetes Cluster MEID not outdated, skipping reconciliation")

		return nil
	}

	k8sconditions.SetStatusOutdated(dk.Conditions(), meIDConditionType, "Kubernetes Cluster MEID is outdated in the status")

	k8sEntity, err := dtc.GetK8sClusterME(ctx, dk.Status.KubeSystemUUID)
	if err != nil {
		log.Info("failed to retrieve MEs")

		return fmt.Errorf("get kubernetes cluster monitored entity: %w", err)
	}

	// in the case the setting was deleted on the tenant, this should be respected in the DK
	dk.Status.KubernetesClusterMEID = k8sEntity.ID
	dk.Status.KubernetesClusterName = k8sEntity.Name

	if k8sEntity.ID == "" {
		log.Info("no MEs found, no kubernetesClusterMEID will be set in the dynakube status")

		return nil
	}

	k8sconditions.SetStatusUpdated(dk.Conditions(), meIDConditionType, "Kubernetes Cluster MEID is up to date")

	log.Info("kubernetesClusterMEID set in dynakube status, done reconciling", "kubernetesClusterMEID", dk.Status.KubernetesClusterMEID)

	return nil
}

// refreshClusterMEID unconditionally fetches and stores the Kubernetes Cluster Monitored Entity ID.
// Used after settings creation to avoid waiting for the next reconciliation cycle.
// Retries up to maxRefreshRetries times with retryInterval between attempts, because the ME
// may not be available immediately after the settings object is created.
func (r *Reconciler) refreshClusterMEID(ctx context.Context, dtc settings.APIClient, dk *dynakube.DynaKube) error {
	const (
		maxRefreshRetries = 5
		retryInterval     = 2 * time.Second
	)

	for attempt := range maxRefreshRetries {
		log.Info("refreshing kubernetes cluster MEID", "attempt", attempt)

		k8sEntity, err := dtc.GetK8sClusterME(ctx, dk.Status.KubeSystemUUID)
		if err != nil {
			return fmt.Errorf("get kubernetes cluster monitored entity: %w", err)
		}

		if k8sEntity.ID != "" {
			dk.Status.KubernetesClusterMEID = k8sEntity.ID
			dk.Status.KubernetesClusterName = k8sEntity.Name
			k8sconditions.SetStatusUpdated(dk.Conditions(), meIDConditionType, "Kubernetes Cluster MEID is up to date")

			log.Info("kubernetesClusterMEID refreshed after settings creation", "kubernetesClusterMEID", dk.Status.KubernetesClusterMEID)

			return nil
		}

		log.Info("kubernetes cluster MEID not yet available after settings creation, will retry", "attempt", attempt)

		if attempt < maxRefreshRetries {
			time.Sleep(retryInterval)
		}
	}

	return goerrors.New("kubernetes cluster MEID not available after settings creation")
}

func (r *Reconciler) createK8sConnectionSettingIfAbsent(ctx context.Context, dtc settings.APIClient, dk *dynakube.DynaKube) (string, error) {
	if dk.Status.KubernetesClusterMEID != "" {
		log.Info("kubernetes cluster setting already exists", "meID", dk.Status.KubernetesClusterMEID, "clusterLabel", dk.Status.KubernetesClusterName, "cluster", dk.Status.KubeSystemUUID)

		return "", nil // settings already exist => don't need to create, and we do not update
	}

	clusterLabel := dk.FF().GetAutomaticK8sAPIMonitoringClusterName()
	if clusterLabel == "" {
		clusterLabel = dk.Name
	}

	objectID, err := dtc.CreateOrUpdateKubernetesSetting(ctx, clusterLabel, dk.Status.KubeSystemUUID, dk.Status.KubernetesClusterMEID)
	if err != nil {
		return "", errors.WithMessage(err, "error creating dynatrace settings object")
	}

	log.Info("created kubernetes cluster setting", "clusterLabel", clusterLabel, "cluster", dk.Status.KubeSystemUUID, "object id", objectID)

	return objectID, nil
}

func (r *Reconciler) createK8sAppSettingIfAbsent(ctx context.Context, dtc settings.APIClient, dk *dynakube.DynaKube) error {
	k8sEntity := settings.K8sClusterME{ID: dk.Status.KubernetesClusterMEID, Name: dk.Status.KubernetesClusterName}
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

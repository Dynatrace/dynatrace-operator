package apimonitoring

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/pkg/errors"
)

type Reconciler struct {
	dtc            dtclient.Client
	clusterLabel   string
	kubeSystemUUID string
}

func NewReconciler(dtc dtclient.Client, clusterLabel, kubeSystemUUID string) *Reconciler {
	return &Reconciler{
		dtc,
		clusterLabel,
		kubeSystemUUID,
	}
}

const (
	settingsSchemaId      = "builtin:cloud.kubernetes"
	appTransitionSchemaId = "builtin:app-transition.kubernetes"
)

func (r *Reconciler) Reconcile(dynakube *dynatracev1beta1.DynaKube) error {
	objectID, err := r.createObjectIdIfNotExists(dynakube)

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

func (r *Reconciler) createObjectIdIfNotExists(dynakube *dynatracev1beta1.DynaKube) (string, error) {
	if r.kubeSystemUUID == "" {
		return "", errors.New("no kube-system namespace UUID given")
	}

	// check if ME with UID exists
	var monitoredEntities, err = r.dtc.GetMonitoredEntitiesForKubeSystemUUID(r.kubeSystemUUID)
	if err != nil {
		return "", errors.WithMessage(err, "error while loading MEs")
	}

	// check if Setting for ME exists
	settings, err := r.dtc.GetSettingsForMonitoredEntities(monitoredEntities, settingsSchemaId)
	if err != nil {
		return "", errors.WithMessage(err, "error trying to check if setting exists")
	}

	if settings.TotalCount > 0 {
		_, err = r.handleKubernetesAppEnabled(dynakube, monitoredEntities)
		if err != nil {
			return "", err
		}
		return "", nil
	}

	// determine newest ME (can be empty string), and create or update a settings object accordingly
	meID := determineNewestMonitoredEntity(monitoredEntities)
	objectID, err := r.dtc.CreateOrUpdateKubernetesSetting(r.clusterLabel, r.kubeSystemUUID, meID)
	if err != nil {
		return "", errors.WithMessage(err, "error creating dynatrace settings object")
	}
	return objectID, nil
}

func (r *Reconciler) handleKubernetesAppEnabled(dynakube *dynatracev1beta1.DynaKube, monitoredEntities []dtclient.MonitoredEntity) (string, error) {
	if dynakube.FeatureEnableK8sAppEnabled() {
		appSettings, err := r.dtc.GetSettingsForMonitoredEntities(monitoredEntities, appTransitionSchemaId)
		if err != nil {
			return "", errors.WithMessage(err, "error trying to check if app setting exists")
		}
		if appSettings.TotalCount == 0 {
			meID := determineNewestMonitoredEntity(monitoredEntities)
			if meID != "" {
				transitionSchemaObjectID, err := r.dtc.CreateOrUpdateKubernetesAppSetting(meID)
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

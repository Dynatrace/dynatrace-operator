package apimonitoring

import (
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

func (r *Reconciler) Reconcile() error {
	objectID, err := r.createObjectIdIfNotExists()

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

func (r *Reconciler) createObjectIdIfNotExists() (string, error) {
	if r.kubeSystemUUID == "" {
		return "", errors.New("no kube-system namespace UUID given")
	}

	// check if ME with UID exists
	var monitoredEntities, err = r.dtc.GetMonitoredEntitiesForKubeSystemUUID(r.kubeSystemUUID)
	if err != nil {
		return "", errors.WithMessage(err, "error while loading MEs")
	}

	// check if Setting for ME exists
	settings, err := r.dtc.GetSettingsForMonitoredEntities(monitoredEntities)
	if err != nil {
		return "", errors.WithMessage(err, "error trying to check if setting exists")
	}

	if settings.TotalCount > 0 {
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

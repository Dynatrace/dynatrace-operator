package automaticapimonitoring

import (
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/pkg/errors"
)

type AutomaticApiMonitoringReconciler struct {
	dtc            dtclient.Client
	name           string
	kubeSystemUUID string
}

func NewReconciler(dtc dtclient.Client, name, kubeSystemUUID string) *AutomaticApiMonitoringReconciler {
	return &AutomaticApiMonitoringReconciler{
		dtc,
		name,
		kubeSystemUUID,
	}
}

func (r *AutomaticApiMonitoringReconciler) Reconcile() error {
	objectID, err := r.ensureSettingExists()

	if err != nil {
		return err
	}

	if objectID != "" {
		log.Info("created kubernetes cluster setting", "name", r.name, "cluster", r.kubeSystemUUID, "object id", objectID)
	} else {
		log.Info("kubernetes cluster setting already exists", "name", r.name, "cluster", r.kubeSystemUUID)
	}

	return nil
}

func (r *AutomaticApiMonitoringReconciler) ensureSettingExists() (string, error) {
	if r.kubeSystemUUID == "" {
		return "", errors.New("no kube-system namespace UUID given")
	}

	// check if ME with UID exists
	var monitoredEntities, err = r.dtc.GetMonitoredEntitiesForKubeSystemUUID(r.kubeSystemUUID)
	if err != nil {
		return "", fmt.Errorf("error while loading MEs: %s", err.Error())
	}

	// check if Setting for ME exists
	settings, err := r.dtc.GetSettingsForMonitoredEntities(monitoredEntities)
	if err != nil {
		return "", fmt.Errorf("error trying to check if setting exists %s", err.Error())
	}

	if settings.TotalCount > 0 {
		return "", nil
	}

	// determine newest ME (can be empty string), and create or update a settings object accordingly
	meID := determineNewestMonitoredEntity(monitoredEntities)
	objectID, err := r.dtc.CreateOrUpdateKubernetesSetting(r.name, r.kubeSystemUUID, meID)

	if err != nil {
		return "", err
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

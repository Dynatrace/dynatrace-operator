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
		log.Info(fmt.Sprintf("created setting '%s' for Cluster '%s'. Settings object ID: %s", r.name, r.kubeSystemUUID, objectID))
	} else {
		log.Info(fmt.Sprintf("setting '%s' for Cluster '%s' already exists.", r.name, r.kubeSystemUUID))
	}

	return nil
}

func (r *AutomaticApiMonitoringReconciler) ensureSettingExists() (string, error) {
	if r.name == "" {
		return "", errors.New("no name given")
	}
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

	var objectID string
	if len(monitoredEntities) > 0 {
		// determine newest me
		meID := determineNewestMonitoredEntity(monitoredEntities)
		objectID, err = r.dtc.CreateKubernetesSetting(r.name, r.kubeSystemUUID, meID)
	} else {
		objectID, err = r.dtc.CreateKubernetesSetting(r.name, r.kubeSystemUUID, "")
	}

	if err != nil {
		return "", err
	}

	return objectID, nil
}

func determineNewestMonitoredEntity(entities []dtclient.MonitoredEntity) string {
	var newestMe dtclient.MonitoredEntity
	for _, entity := range entities {
		if entity.LastSeenTms > newestMe.LastSeenTms {
			newestMe = entity
		}
	}

	return newestMe.EntityId
}

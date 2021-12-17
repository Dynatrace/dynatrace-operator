package automaticapimonitoring

import (
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/status"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/pkg/errors"
)

type Reconciler struct {
	dtclient.Client
}

func NewReconciler(clt dtclient.Client) *Reconciler {
	return &Reconciler{
		clt,
	}
}

func (r *Reconciler) Reconcile(dkState *status.DynakubeState) error {
	name := dkState.Instance.Name
	kubeSystemUUID := dkState.Instance.Status.KubeSystemUUID
	objectID, err := r.ensureSettingExists(r.Client, name, kubeSystemUUID)

	if err != nil {
		return err
	}

	if objectID != "" {
		log.Info(fmt.Sprintf("created setting '%s' for Cluster '%s'. Settings object ID: %s", name, kubeSystemUUID, objectID))
	} else {
		log.Info(fmt.Sprintf("setting '%s' for Cluster '%s' already exists.", name, kubeSystemUUID))
	}

	return nil
}

func (r *Reconciler) ensureSettingExists(dtc dtclient.Client, name string, kubeSystemUUID string) (string, error) {
	if name == "" {
		return "", errors.New("no name given")
	}
	if kubeSystemUUID == "" {
		return "", errors.New("no kube-system namespace UUID given")
	}

	// check if ME with UID exists
	var monitoredEntities, err = dtc.GetMonitoredEntitiesForKubeSystemUUID(kubeSystemUUID)
	if err != nil {
		return "", fmt.Errorf("error while loading MEs: %s", err.Error())
	}

	// check if Setting for ME exists
	settings, err := dtc.GetSettingsForMonitoredEntities(monitoredEntities)
	if err != nil {
		return "", fmt.Errorf("error trying to check if setting excists %s", err.Error())
	}

	if settings.TotalCount > 0 {
		return "", nil
	}

	var objectID string
	if len(monitoredEntities) > 0 {
		// determine newest me
		meID := determineNewestMonitoredEntity(monitoredEntities)
		objectID, err = dtc.CreateKubernetesSetting(name, kubeSystemUUID, meID)
	} else {
		objectID, err = dtc.CreateKubernetesSetting(name, kubeSystemUUID, "")
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

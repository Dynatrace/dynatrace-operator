package connectioninfo

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *Reconciler) reconcileActiveGateConnectionInfo() error {
	if !r.dynakube.IsActiveGateConnectionInfoUpdateAllowed(r.timeProvider) {
		log.Info(dynatracev1beta1.GetCacheValidMessage(
			"activegate connection info update",
			r.dynakube.Status.ActiveGate.ConnectionInfoStatus.LastRequest,
			r.dynakube.FeatureApiRequestThreshold()))
		return nil
	}

	connectionInfo, err := r.dtc.GetActiveGateConnectionInfo()
	if err != nil {
		log.Info("failed to get activegate connection info")
		return err
	}

	err = r.maintainActiveGateConnectionInfoObjects(connectionInfo)
	if err != nil {
		return err
	}

	log.Info("activegate connection info updated")
	r.dynakube.Status.ActiveGate.ConnectionInfoStatus.LastRequest = metav1.Now()
	return nil
}

func (r *Reconciler) maintainActiveGateConnectionInfoObjects(connectionInfo dtclient.ActiveGateConnectionInfo) error {
	err := r.createTenantTokenSecret(r.dynakube.ActivegateTenantSecret(), connectionInfo.ConnectionInfo)
	if err != nil {
		return err
	}

	err = r.createActiveGateTenantConnectionInfoConfigMap(r.dynakube.ActiveGateConnectionInfoConfigMapName(), connectionInfo)
	if err != nil {
		return err
	}

	r.updateDynakubeActiveGateStatus(connectionInfo)
	return nil
}

func (r *Reconciler) createActiveGateTenantConnectionInfoConfigMap(secretName string, connectionInfo dtclient.ActiveGateConnectionInfo) error {
	configMapData := extractPublicData(connectionInfo.ConnectionInfo)
	configMap, err := kubeobjects.CreateConfigMap(r.scheme, r.dynakube,
		kubeobjects.NewConfigMapNameModifier(secretName),
		kubeobjects.NewConfigMapNamespaceModifier(r.dynakube.Namespace),
		kubeobjects.NewConfigMapDataModifier(configMapData))
	if err != nil {
		return errors.WithStack(err)
	}

	query := kubeobjects.NewConfigMapQuery(r.context, r.client, r.apiReader, log)
	err = query.CreateOrUpdate(*configMap)
	if err != nil {
		log.Info("could not create or update configMap for connection info", "name", configMap.Name)
		return err
	}
	return nil
}

func (r *Reconciler) updateDynakubeActiveGateStatus(connectionInfo dtclient.ActiveGateConnectionInfo) {
	r.dynakube.Status.ActiveGate.ConnectionInfoStatus.TenantUUID = connectionInfo.TenantUUID
	r.dynakube.Status.ActiveGate.ConnectionInfoStatus.TenantToken = connectionInfo.TenantToken
	r.dynakube.Status.ActiveGate.ConnectionInfoStatus.Endpoints = connectionInfo.Endpoints
}

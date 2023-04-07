package connectioninfo

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *Reconciler) reconcileOneAgentConnectionInfo() error {
	if !r.dynakube.IsOneAgentConnectionInfoUpdateAllowed(r.timeProvider) {
		log.Info(dynatracev1beta1.GetCacheValidMessage(
			"oneagent connection info update",
			r.dynakube.Status.OneAgent.ConnectionInfoStatus.LastRequest,
			r.dynakube.FeatureApiRequestThreshold()))
		return nil
	}

	connectionInfo, err := r.dtc.GetOneAgentConnectionInfo()
	if err != nil {
		log.Info("failed to get oneagent connection info")
		return err
	}

	err = r.maintainOneAgentConnectionInfoObjects(connectionInfo)
	if err != nil {
		return err
	}

	log.Info("oneagent connection info updated")

	r.dynakube.Status.OneAgent.ConnectionInfoStatus.LastRequest = metav1.Now()
	return nil
}

func (r *Reconciler) maintainOneAgentConnectionInfoObjects(connectionInfo dtclient.OneAgentConnectionInfo) error {
	err := r.createTenantTokenSecret(r.dynakube.OneagentTenantSecret(), connectionInfo.ConnectionInfo)
	if err != nil {
		return err
	}

	err = r.createOneAgentTenantConnectionInfoConfigMap(r.dynakube.OneAgentConnectionInfoConfigMapName(), connectionInfo)
	if err != nil {
		return err
	}

	r.updateDynakubeOneAgentStatus(connectionInfo)

	return nil
}

func (r *Reconciler) createOneAgentTenantConnectionInfoConfigMap(configMapName string, connectionInfo dtclient.OneAgentConnectionInfo) error {
	configMapData := extractPublicData(connectionInfo.ConnectionInfo)
	configMap, err := kubeobjects.CreateConfigMap(r.scheme, r.dynakube,
		kubeobjects.NewConfigMapNameModifier(configMapName),
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

func (r *Reconciler) updateDynakubeOneAgentStatus(connectionInfo dtclient.OneAgentConnectionInfo) {
	r.dynakube.Status.OneAgent.ConnectionInfoStatus.TenantUUID = connectionInfo.TenantUUID
	r.dynakube.Status.OneAgent.ConnectionInfoStatus.TenantToken = connectionInfo.TenantToken
	r.dynakube.Status.OneAgent.ConnectionInfoStatus.Endpoints = connectionInfo.Endpoints
	copyCommunicationHosts(r.dynakube.Status.OneAgent.ConnectionInfoStatus, connectionInfo.CommunicationHosts)
}

func copyCommunicationHosts(dest dynatracev1beta1.OneAgentConnectionInfoStatus, src []dtclient.CommunicationHost) {
	if dest.CommunicationHosts == nil {
		dest.CommunicationHosts = make([]dynatracev1beta1.CommunicationHostStatus, 0, len(src))
	}
	for _, host := range src {
		dest.CommunicationHosts = append(dest.CommunicationHosts, dynatracev1beta1.CommunicationHostStatus{
			Protocol: host.Protocol,
			Host:     host.Host,
			Port:     host.Port,
		})
	}
}

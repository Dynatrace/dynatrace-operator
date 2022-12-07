package connectioninfo

import (
	"context"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler struct {
	context   context.Context
	client    client.Client
	apiReader client.Reader
	dtc       dtclient.Client
	dynakube  *dynatracev1beta1.DynaKube
}

func NewReconciler(ctx context.Context, clt client.Client, apiReader client.Reader, dynakube *dynatracev1beta1.DynaKube, dtc dtclient.Client) *Reconciler { //nolint:revive // argument-limit doesn't apply to constructors
	return &Reconciler{
		context:   ctx,
		client:    clt,
		apiReader: apiReader,
		dynakube:  dynakube,
		dtc:       dtc,
	}
}

func (r *Reconciler) Reconcile() error {
	if !r.dynakube.FeatureDisableActivegateRawImage() {
		err := r.reconcileActiveGateConnectionInfo()
		if err != nil {
			return err
		}
	}

	err := r.reconcileOneAgentConnectionInfo()
	if err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) reconcileOneAgentConnectionInfo() error {
	oneAgentConnectionInfo, err := r.dtc.GetOneAgentConnectionInfo()
	if err != nil {
		log.Info("failed to get oneagent connection info")
		return err
	}

	err = r.maintainConnectionInfoObjects(r.dynakube.OneagentTenantSecret(), r.dynakube.OneAgentConnectionInfoConfigMapName(), oneAgentConnectionInfo.ConnectionInfo)
	if err != nil {
		return err
	}
	return nil
}

func (r *Reconciler) reconcileActiveGateConnectionInfo() error {
	activeGateConnectionInfo, err := r.dtc.GetActiveGateConnectionInfo()
	if err != nil {
		log.Info("failed to get activegate connection info")
		return err
	}

	err = r.maintainConnectionInfoObjects(r.dynakube.ActivegateTenantSecret(), r.dynakube.ActiveGateConnectionInfoConfigMapName(), activeGateConnectionInfo.ConnectionInfo)
	if err != nil {
		return err
	}
	return nil
}

func (r *Reconciler) maintainConnectionInfoObjects(secretName string, configMapName string, connectionInfo dtclient.ConnectionInfo) error {
	err := r.createTokenSecret(secretName, connectionInfo)
	if err != nil {
		return err
	}

	err = r.createTenantUuidAndCommunicationEndpointsConfigMap(configMapName, connectionInfo)
	if err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) createTenantUuidAndCommunicationEndpointsConfigMap(secretName string, connectionInfo dtclient.ConnectionInfo) error {
	configMapData := buildConnectionInfoConfigMap(connectionInfo)
	configMap := kubeobjects.NewConfigMap(secretName, r.dynakube.Namespace, configMapData)

	query := kubeobjects.NewConfigMapQuery(r.context, r.client, r.apiReader, log)
	err := query.CreateOrUpdate(*configMap)
	if err != nil {
		log.Info("could not create or update configMap for connection info", "name", configMap.Name)
		return err
	}
	return nil
}

func (r *Reconciler) createTokenSecret(secretName string, connectionInfo dtclient.ConnectionInfo) error {
	secretData := buildConnectionInfoSecret(connectionInfo)
	secret := kubeobjects.NewSecret(secretName, r.dynakube.Namespace, secretData)

	query := kubeobjects.NewSecretQuery(r.context, r.client, r.apiReader, log)
	err := query.CreateOrUpdate(*secret)
	if err != nil {
		log.Info("could not create or update secret for connection info", "name", secret.Name)
		return err
	}
	return nil
}

func buildConnectionInfoSecret(connectionInfo dtclient.ConnectionInfo) map[string][]byte {
	data := map[string][]byte{
		TenantTokenName: []byte(connectionInfo.TenantToken),
	}

	return data
}

func buildConnectionInfoConfigMap(connectionInfo dtclient.ConnectionInfo) map[string]string {
	data := map[string]string{}

	if connectionInfo.TenantUUID != "" {
		data[REMOVE_IT_TenantUuidName] = connectionInfo.TenantUUID
	}
	if connectionInfo.Endpoints != "" {
		data[CommunicationEndpointsName] = connectionInfo.Endpoints
	}

	return data
}

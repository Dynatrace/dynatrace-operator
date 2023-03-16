package connectioninfo

import (
	"context"
	"encoding/json"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/timeprovider"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler struct {
	context      context.Context
	client       client.Client
	apiReader    client.Reader
	dtc          dtclient.Client
	dynakube     *dynatracev1beta1.DynaKube
	scheme       *runtime.Scheme
	timeProvider *timeprovider.Provider
}

func NewReconciler(ctx context.Context, clt client.Client, apiReader client.Reader, scheme *runtime.Scheme, dynakube *dynatracev1beta1.DynaKube, dtc dtclient.Client) *Reconciler { //nolint:revive // argument-limit doesn't apply to constructors
	return &Reconciler{
		context:      ctx,
		client:       clt,
		apiReader:    apiReader,
		dynakube:     dynakube,
		scheme:       scheme,
		dtc:          dtc,
		timeProvider: timeprovider.New(),
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
	if !r.dynakube.IsOneAgentConnectionInfoUpdateAllowed(r.timeProvider) {
		log.Info(dynatracev1beta1.GetCacheValidMessage(
			"oneagent connection info update",
			r.dynakube.Status.DynatraceApi.LastOneAgentConnectionInfoRequest,
			r.dynakube.FeatureApiRequestThreshold()))
		return nil
	}

	err := r.maintainOneAgentConnectionInfoObjects()
	if err != nil {
		return err
	}

	r.dynakube.Status.DynatraceApi.LastOneAgentConnectionInfoRequest = metav1.Now()
	log.Info("oneagent connection info updated")
	return nil
}

func (r *Reconciler) reconcileActiveGateConnectionInfo() error {
	if !r.dynakube.IsActiveGateConnectionInfoUpdateAllowed(r.timeProvider) {
		log.Info(dynatracev1beta1.GetCacheValidMessage(
			"activegate connection info update",
			r.dynakube.Status.DynatraceApi.LastActiveGateConnectionInfoRequest,
			r.dynakube.FeatureApiRequestThreshold()))
		return nil
	}

	err := r.maintainActiveGateConnectionInfoObjects()
	if err != nil {
		return err
	}

	r.dynakube.Status.DynatraceApi.LastActiveGateConnectionInfoRequest = metav1.Now()

	log.Info("activegate connection info updated")
	return nil
}

func (r *Reconciler) maintainActiveGateConnectionInfoObjects() error {
	connectionInfo, err := r.dtc.GetActiveGateConnectionInfo()
	if err != nil {
		log.Info("failed to get activegate connection info")
		return err
	}

	err = r.createTenantTokenSecret(r.dynakube.ActivegateTenantSecret(), connectionInfo.ConnectionInfo)
	if err != nil {
		return err
	}

	err = r.createActiveGateTenantConnectionInfoConfigMap(r.dynakube.ActiveGateConnectionInfoConfigMapName(), connectionInfo.ConnectionInfo)
	if err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) maintainOneAgentConnectionInfoObjects() error {
	connectionInfo, err := r.dtc.GetOneAgentConnectionInfo()
	if err != nil {
		log.Info("failed to get oneagent connection info")
		return err
	}

	err = r.createTenantTokenSecret(r.dynakube.OneagentTenantSecret(), connectionInfo.ConnectionInfo)
	if err != nil {
		return err
	}

	err = r.createOneAgentTenantConnectionInfoConfigMap(r.dynakube.OneAgentConnectionInfoConfigMapName(), connectionInfo)
	if err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) createActiveGateTenantConnectionInfoConfigMap(secretName string, connectionInfo dtclient.ConnectionInfo) error {
	configMapData := extractPublicData(connectionInfo, "")
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

func (r *Reconciler) createOneAgentTenantConnectionInfoConfigMap(secretName string, connectionInfo dtclient.OneAgentConnectionInfo) error {
	communicationHosts, err := encodeCommunicationHosts(connectionInfo)
	if err != nil {
		return err
	}
	configMapData := extractPublicData(connectionInfo.ConnectionInfo, communicationHosts)
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

func (r *Reconciler) createTenantTokenSecret(secretName string, connectionInfo dtclient.ConnectionInfo) error {
	secretData := extractSensitiveData(connectionInfo)
	secret, err := kubeobjects.CreateSecret(r.scheme, r.dynakube,
		kubeobjects.NewSecretNameModifier(secretName),
		kubeobjects.NewSecretNamespaceModifier(r.dynakube.Namespace),
		kubeobjects.NewSecretDataModifier(secretData))
	if err != nil {
		return errors.WithStack(err)
	}

	query := kubeobjects.NewSecretQuery(r.context, r.client, r.apiReader, log)
	err = query.CreateOrUpdate(*secret)
	if err != nil {
		log.Info("could not create or update secret for connection info", "name", secret.Name)
		return err
	}
	return nil
}

func extractSensitiveData(connectionInfo dtclient.ConnectionInfo) map[string][]byte {
	data := map[string][]byte{
		TenantTokenName: []byte(connectionInfo.TenantToken),
	}

	return data
}

func extractPublicData(connectionInfo dtclient.ConnectionInfo, communicationHosts string) map[string]string {
	data := map[string]string{}

	if connectionInfo.TenantUUID != "" {
		data[TenantUUIDName] = connectionInfo.TenantUUID
	}
	if connectionInfo.Endpoints != "" {
		data[CommunicationEndpointsName] = connectionInfo.Endpoints
	}

	if communicationHosts != "" {
		data[CommunicationHosts] = communicationHosts
	}
	return data
}

func encodeCommunicationHosts(connectionInfo dtclient.OneAgentConnectionInfo) (string, error) {
	if len(connectionInfo.CommunicationHosts) > 0 {
		communicationHostsBytes, err := json.Marshal(connectionInfo.CommunicationHosts)
		if err != nil {
			return "", errors.WithStack(err)
		}
		return string(communicationHostsBytes), nil
	}
	return "", nil
}

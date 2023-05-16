package connectioninfo

import (
	"context"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/timeprovider"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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

func (r *Reconciler) needsUpdate(secretName string, isAllowedFunc dynatracev1beta1.RequestAllowedChecker) (bool, error) {
	query := kubeobjects.NewSecretQuery(r.context, r.client, r.apiReader, log)
	_, err := query.Get(types.NamespacedName{Name: secretName, Namespace: r.dynakube.Namespace})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("creating secret, because missing", "secretName", secretName)
			return true, nil
		}
		return false, err
	}
	return isAllowedFunc(r.timeProvider), nil
}

func (r *Reconciler) reconcileOneAgentConnectionInfo() error {
	needsUpdate, err := r.needsUpdate(r.dynakube.OneagentTenantSecret(), r.dynakube.IsOneAgentConnectionInfoUpdateAllowed)
	if err != nil {
		return err
	}
	if !needsUpdate {
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

	r.updateDynakubeOneAgentStatus(connectionInfo)

	err = r.createTenantTokenSecret(r.dynakube.OneagentTenantSecret(), connectionInfo.ConnectionInfo)
	if err != nil {
		return err
	}

	log.Info("oneagent connection info updated")
	r.dynakube.Status.OneAgent.ConnectionInfoStatus.LastRequest = metav1.Now()
	return nil
}

func (r *Reconciler) updateDynakubeOneAgentStatus(connectionInfo dtclient.OneAgentConnectionInfo) {
	r.dynakube.Status.OneAgent.ConnectionInfoStatus.TenantUUID = connectionInfo.TenantUUID
	r.dynakube.Status.OneAgent.ConnectionInfoStatus.Endpoints = connectionInfo.Endpoints
	copyCommunicationHosts(&r.dynakube.Status.OneAgent.ConnectionInfoStatus, connectionInfo.CommunicationHosts)
}

func copyCommunicationHosts(dest *dynatracev1beta1.OneAgentConnectionInfoStatus, src []dtclient.CommunicationHost) {
	dest.CommunicationHosts = make([]dynatracev1beta1.CommunicationHostStatus, 0, len(src))
	for _, host := range src {
		dest.CommunicationHosts = append(dest.CommunicationHosts, dynatracev1beta1.CommunicationHostStatus{
			Protocol: host.Protocol,
			Host:     host.Host,
			Port:     host.Port,
		})
	}
}

func (r *Reconciler) reconcileActiveGateConnectionInfo() error {
	needsUpdate, err := r.needsUpdate(r.dynakube.ActivegateTenantSecret(), r.dynakube.IsActiveGateConnectionInfoUpdateAllowed)
	if err != nil {
		return err
	}
	if !needsUpdate {
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

	r.updateDynakubeActiveGateStatus(connectionInfo)

	err = r.createTenantTokenSecret(r.dynakube.ActivegateTenantSecret(), connectionInfo.ConnectionInfo)
	if err != nil {
		return err
	}

	log.Info("activegate connection info updated")
	r.dynakube.Status.ActiveGate.ConnectionInfoStatus.LastRequest = metav1.Now()
	return nil
}

func (r *Reconciler) updateDynakubeActiveGateStatus(connectionInfo dtclient.ActiveGateConnectionInfo) {
	r.dynakube.Status.ActiveGate.ConnectionInfoStatus.TenantUUID = connectionInfo.TenantUUID
	r.dynakube.Status.ActiveGate.ConnectionInfoStatus.Endpoints = connectionInfo.Endpoints
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

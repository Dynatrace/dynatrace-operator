package connectioninfo

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	controllererrors "github.com/Dynatrace/dynatrace-operator/pkg/controllers/errors"
	k8ssecret "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var NoOneAgentCommunicationHostsError = errors.New("no communication hosts for OneAgent are available")
var ConnectionInfoUpdatedNotification = controllererrors.NewRestartReconciliationError("connection info updated, restart required")

type Reconciler struct {
	client       client.Client
	apiReader    client.Reader
	dtc          dtclient.Client
	dynakube     *dynatracev1beta1.DynaKube
	scheme       *runtime.Scheme
	timeProvider *timeprovider.Provider
}

func NewReconciler(clt client.Client, apiReader client.Reader, scheme *runtime.Scheme, dynakube *dynatracev1beta1.DynaKube, dtc dtclient.Client) *Reconciler { //nolint:revive // argument-limit doesn't apply to constructors
	return &Reconciler{
		client:       clt,
		apiReader:    apiReader,
		dynakube:     dynakube,
		scheme:       scheme,
		dtc:          dtc,
		timeProvider: timeprovider.New(),
	}
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	var activeGateConnectionInfoUpdated bool
	if !r.dynakube.FeatureDisableActivegateRawImage() {
		var err error
		activeGateConnectionInfoUpdated, err = r.reconcileActiveGateConnectionInfo(ctx)
		if err != nil {
			return err
		}
	}

	oneAgentConnectionInfoUpdated, err := r.reconcileOneAgentConnectionInfo(ctx)
	if err != nil {
		return err
	}

	if oneAgentConnectionInfoUpdated || activeGateConnectionInfoUpdated {
		return ConnectionInfoUpdatedNotification
	}
	return nil
}

func (r *Reconciler) needsUpdate(ctx context.Context, secretName string, isAllowedFunc dynatracev1beta1.RequestAllowedChecker) (bool, error) {
	query := k8ssecret.NewQuery(ctx, r.client, r.apiReader, log)
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

func (r *Reconciler) reconcileOneAgentConnectionInfo(ctx context.Context) (bool, error) {
	needsUpdate, err := r.needsUpdate(ctx, r.dynakube.OneagentTenantSecret(), r.dynakube.IsOneAgentConnectionInfoUpdateAllowed)
	if err != nil {
		return false, err
	}
	if !needsUpdate {
		log.Info(dynatracev1beta1.GetCacheValidMessage(
			"OneAgent connection info update",
			r.dynakube.Status.OneAgent.ConnectionInfoStatus.LastRequest,
			r.dynakube.FeatureApiRequestThreshold()))
		return false, nil
	}

	connectionInfo, err := r.dtc.GetOneAgentConnectionInfo()
	if err != nil {
		log.Info("failed to get OneAgent connection info")
		return false, err
	}

	r.updateDynakubeOneAgentStatus(connectionInfo)

	err = r.createTenantTokenSecret(ctx, r.dynakube.OneagentTenantSecret(), connectionInfo.ConnectionInfo)
	if err != nil {
		return false, err
	}

	log.Info("OneAgent connection info updated")

	if len(connectionInfo.Endpoints) == 0 {
		log.Info("tenant has no endpoints", "tenant", connectionInfo.TenantUUID)
	}

	if len(connectionInfo.CommunicationHosts) == 0 {
		log.Info("no OneAgent communication hosts received, tenant API requests not yet throttled")
		return false, NoOneAgentCommunicationHostsError
	}

	log.Info("received OneAgent communication hosts", "communication hosts", connectionInfo.CommunicationHosts, "tenant", connectionInfo.TenantUUID)

	r.dynakube.Status.OneAgent.ConnectionInfoStatus.LastRequest = metav1.Now()
	return true, nil
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

func (r *Reconciler) reconcileActiveGateConnectionInfo(ctx context.Context) (bool, error) {
	needsUpdate, err := r.needsUpdate(ctx, r.dynakube.ActivegateTenantSecret(), r.dynakube.IsActiveGateConnectionInfoUpdateAllowed)
	if err != nil {
		return false, err
	}
	if !needsUpdate {
		log.Info(dynatracev1beta1.GetCacheValidMessage(
			"activegate connection info update",
			r.dynakube.Status.ActiveGate.ConnectionInfoStatus.LastRequest,
			r.dynakube.FeatureApiRequestThreshold()))
		return false, nil
	}

	connectionInfo, err := r.dtc.GetActiveGateConnectionInfo()
	if err != nil {
		log.Info("failed to get activegate connection info")
		return false, err
	}

	r.updateDynakubeActiveGateStatus(connectionInfo)

	err = r.createTenantTokenSecret(ctx, r.dynakube.ActivegateTenantSecret(), connectionInfo.ConnectionInfo)
	if err != nil {
		return false, err
	}

	log.Info("activegate connection info updated")
	r.dynakube.Status.ActiveGate.ConnectionInfoStatus.LastRequest = metav1.Now()
	return true, nil
}

func (r *Reconciler) updateDynakubeActiveGateStatus(connectionInfo dtclient.ActiveGateConnectionInfo) {
	r.dynakube.Status.ActiveGate.ConnectionInfoStatus.TenantUUID = connectionInfo.TenantUUID
	r.dynakube.Status.ActiveGate.ConnectionInfoStatus.Endpoints = connectionInfo.Endpoints
}

func (r *Reconciler) createTenantTokenSecret(ctx context.Context, secretName string, connectionInfo dtclient.ConnectionInfo) error {
	secretData := extractSensitiveData(connectionInfo)
	secret, err := k8ssecret.Create(r.scheme, r.dynakube,
		k8ssecret.NewNameModifier(secretName),
		k8ssecret.NewNamespaceModifier(r.dynakube.Namespace),
		k8ssecret.NewDataModifier(secretData))
	if err != nil {
		return errors.WithStack(err)
	}

	query := k8ssecret.NewQuery(ctx, r.client, r.apiReader, log)
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

package connectioninfo

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
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

type Reconciler interface {
	ReconcileActiveGate(ctx context.Context, dynakube *dynatracev1beta1.DynaKube) error
	ReconcileOneAgent(ctx context.Context, dynakube *dynatracev1beta1.DynaKube) error
}

type reconciler struct {
	client       client.Client
	apiReader    client.Reader
	dtc          dtclient.Client
	scheme       *runtime.Scheme
	timeProvider *timeprovider.Provider
}
type ReconcilerBuilder func(clt client.Client, apiReader client.Reader, scheme *runtime.Scheme, dtc dtclient.Client) Reconciler

func NewReconciler(clt client.Client, apiReader client.Reader, scheme *runtime.Scheme, dtc dtclient.Client) Reconciler {
	return &reconciler{
		client:       clt,
		apiReader:    apiReader,
		scheme:       scheme,
		dtc:          dtc,
		timeProvider: timeprovider.New(),
	}
}

var NoOneAgentCommunicationHostsError = errors.New("no communication hosts for OneAgent are available")

func (r *reconciler) ReconcileActiveGate(ctx context.Context, dynakube *dynatracev1beta1.DynaKube) error {
	oldStatus := dynakube.Status.DeepCopy()

	err := r.reconcileActiveGateConnectionInfo(ctx, dynakube)
	if err != nil {
		return err
	}

	needStatusUpdate, err := hasher.IsDifferent(oldStatus, dynakube.Status)
	if err != nil {
		return errors.WithMessage(err, "failed to compare connection info status hashes")
	} else if needStatusUpdate {
		err = dynakube.UpdateStatus(ctx, r.client)
	}

	return err
}

func (r *reconciler) ReconcileOneAgent(ctx context.Context, dynakube *dynatracev1beta1.DynaKube) error {
	oldStatus := dynakube.Status.DeepCopy()

	err := r.reconcileOneAgentConnectionInfo(ctx, dynakube)
	if err != nil {
		return err
	}

	needStatusUpdate, err := hasher.IsDifferent(oldStatus, dynakube.Status)
	if err != nil {
		return errors.WithMessage(err, "failed to compare connection info status hashes")
	} else if needStatusUpdate {
		err = dynakube.UpdateStatus(ctx, r.client)
	}

	return err
}

func (r *reconciler) needsUpdate(ctx context.Context, secretNamespacedName types.NamespacedName, isAllowedFunc dynatracev1beta1.RequestAllowedChecker) (bool, error) {
	query := k8ssecret.NewQuery(ctx, r.client, r.apiReader, log)

	_, err := query.Get(secretNamespacedName)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("creating secret, because missing", "secretName", secretNamespacedName.Name)

			return true, nil
		}

		return false, err
	}

	return isAllowedFunc(r.timeProvider), nil
}

func (r *reconciler) reconcileOneAgentConnectionInfo(ctx context.Context, dynakube *dynatracev1beta1.DynaKube) error {
	needsUpdate, err := r.needsUpdate(ctx, types.NamespacedName{Name: dynakube.OneagentTenantSecret(), Namespace: dynakube.Namespace}, dynakube.IsOneAgentConnectionInfoUpdateAllowed)
	if err != nil {
		return err
	}

	if !needsUpdate {
		log.Info(dynatracev1beta1.GetCacheValidMessage(
			"OneAgent connection info update",
			dynakube.Status.OneAgent.ConnectionInfoStatus.LastRequest,
			dynakube.FeatureApiRequestThreshold()))

		return nil
	}

	connectionInfo, err := r.dtc.GetOneAgentConnectionInfo(ctx)
	if err != nil {
		return errors.WithMessage(err, "failed to get OneAgent connection info")
	}

	r.updateDynakubeOneAgentStatus(dynakube, connectionInfo)

	err = r.createTenantTokenSecret(ctx, dynakube.OneagentTenantSecret(), dynakube, connectionInfo.ConnectionInfo)
	if err != nil {
		return err
	}

	log.Info("OneAgent connection info updated")

	if len(connectionInfo.Endpoints) == 0 {
		log.Info("tenant has no endpoints", "tenant", connectionInfo.TenantUUID)
	}

	if len(connectionInfo.CommunicationHosts) == 0 {
		log.Info("no OneAgent communication hosts received, tenant API requests not yet throttled")

		return NoOneAgentCommunicationHostsError
	}

	log.Info("received OneAgent communication hosts", "communication hosts", connectionInfo.CommunicationHosts, "tenant", connectionInfo.TenantUUID)

	dynakube.Status.OneAgent.ConnectionInfoStatus.LastRequest = metav1.Now()

	return nil
}

func (r *reconciler) updateDynakubeOneAgentStatus(dynakube *dynatracev1beta1.DynaKube, connectionInfo dtclient.OneAgentConnectionInfo) {
	dynakube.Status.OneAgent.ConnectionInfoStatus.TenantUUID = connectionInfo.TenantUUID
	dynakube.Status.OneAgent.ConnectionInfoStatus.Endpoints = connectionInfo.Endpoints
	copyCommunicationHosts(&dynakube.Status.OneAgent.ConnectionInfoStatus, connectionInfo.CommunicationHosts)
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

func (r *reconciler) reconcileActiveGateConnectionInfo(ctx context.Context, dynakube *dynatracev1beta1.DynaKube) error {
	needsUpdate, err := r.needsUpdate(ctx, types.NamespacedName{Name: dynakube.ActivegateTenantSecret(), Namespace: dynakube.Namespace}, dynakube.IsActiveGateConnectionInfoUpdateAllowed)
	if err != nil {
		return err
	}

	if !needsUpdate {
		log.Info(dynatracev1beta1.GetCacheValidMessage(
			"activegate connection info update",
			dynakube.Status.ActiveGate.ConnectionInfoStatus.LastRequest,
			dynakube.FeatureApiRequestThreshold()))

		return nil
	}

	connectionInfo, err := r.dtc.GetActiveGateConnectionInfo(ctx)
	if err != nil {
		log.Info("failed to get activegate connection info")

		return err
	}

	r.updateDynakubeActiveGateStatus(dynakube, connectionInfo)

	err = r.createTenantTokenSecret(ctx, dynakube.ActivegateTenantSecret(), dynakube, connectionInfo.ConnectionInfo)
	if err != nil {
		return err
	}

	log.Info("activegate connection info updated")

	dynakube.Status.ActiveGate.ConnectionInfoStatus.LastRequest = metav1.Now()

	return nil
}

func (r *reconciler) updateDynakubeActiveGateStatus(dynakube *dynatracev1beta1.DynaKube, connectionInfo dtclient.ActiveGateConnectionInfo) {
	dynakube.Status.ActiveGate.ConnectionInfoStatus.TenantUUID = connectionInfo.TenantUUID
	dynakube.Status.ActiveGate.ConnectionInfoStatus.Endpoints = connectionInfo.Endpoints
}

func (r *reconciler) createTenantTokenSecret(ctx context.Context, secretName string, owner metav1.Object, connectionInfo dtclient.ConnectionInfo) error {
	secretData := extractSensitiveData(connectionInfo)

	secret, err := k8ssecret.Create(r.scheme, owner,
		k8ssecret.NewNameModifier(secretName),
		k8ssecret.NewNamespaceModifier(owner.GetNamespace()),
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

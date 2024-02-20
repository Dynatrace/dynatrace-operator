package oaconnectioninfo

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	k8ssecret "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type reconciler struct {
	client       client.Client
	apiReader    client.Reader
	dtc          dtclient.Client
	scheme       *runtime.Scheme
	timeProvider *timeprovider.Provider

	dynakube *dynatracev1beta1.DynaKube
}
type ReconcilerBuilder func(clt client.Client, apiReader client.Reader, scheme *runtime.Scheme, dtc dtclient.Client, dynakube *dynatracev1beta1.DynaKube) controllers.Reconciler

var _ ReconcilerBuilder = NewReconciler

func NewReconciler(clt client.Client, apiReader client.Reader, scheme *runtime.Scheme, dtc dtclient.Client, dynakube *dynatracev1beta1.DynaKube) controllers.Reconciler {
	return &reconciler{
		client:       clt,
		apiReader:    apiReader,
		dynakube:     dynakube,
		scheme:       scheme,
		dtc:          dtc,
		timeProvider: timeprovider.New(),
	}
}

var NoOneAgentCommunicationHostsError = errors.New("no communication hosts for OneAgent are available")

func (r *reconciler) Reconcile(ctx context.Context) error {
	oldStatus := r.dynakube.Status.DeepCopy()

	err := r.reconcileConnectionInfo(ctx)
	if err != nil {
		return err
	}

	needStatusUpdate, err := hasher.IsDifferent(oldStatus, r.dynakube.Status)
	if err != nil {
		return errors.WithMessage(err, "failed to compare connection info status hashes")
	} else if needStatusUpdate {
		err = r.dynakube.UpdateStatus(ctx, r.client)
	}

	return err
}

func (r *reconciler) reconcileConnectionInfo(ctx context.Context) error {
	secretNamespacedName := types.NamespacedName{Name: r.dynakube.OneagentTenantSecret(), Namespace: r.dynakube.Namespace}
	isOutdated := r.dynakube.IsOneAgentConnectionInfoUpdateAllowed(r.timeProvider)

	if !isOutdated {
		needsUpdate, err := connectioninfo.SecretNotPresent(ctx, r.apiReader, secretNamespacedName, log)
		if err != nil {
			return err
		}

		if !needsUpdate {
			log.Info(dynatracev1beta1.GetCacheValidMessage(
				"OneAgent connection info update",
				r.dynakube.Status.OneAgent.ConnectionInfoStatus.LastRequest,
				r.dynakube.FeatureApiRequestThreshold()))

			return nil
		}
	}

	connectionInfo, err := r.dtc.GetOneAgentConnectionInfo(ctx)
	if err != nil {
		return errors.WithMessage(err, "failed to get OneAgent connection info")
	}

	r.setDynakubeStatus(connectionInfo)

	err = r.createTenantTokenSecret(ctx, r.dynakube.OneagentTenantSecret(), r.dynakube, connectionInfo.ConnectionInfo)
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

	r.dynakube.Status.OneAgent.ConnectionInfoStatus.LastRequest = metav1.Now()

	return nil
}

func (r *reconciler) setDynakubeStatus(connectionInfo dtclient.OneAgentConnectionInfo) {
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

// TODO: Generalize
func (r *reconciler) createTenantTokenSecret(ctx context.Context, secretName string, owner metav1.Object, connectionInfo dtclient.ConnectionInfo) error {
	secretData := connectioninfo.ExtractSensitiveData(connectionInfo)

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

package oaconnectioninfo

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	k8ssecret "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type reconciler struct {
	client       client.Client
	apiReader    client.Reader
	dtc          dtclient.Client
	timeProvider *timeprovider.Provider

	dk *dynakube.DynaKube
}
type ReconcilerBuilder func(clt client.Client, apiReader client.Reader, dtc dtclient.Client, dk *dynakube.DynaKube) controllers.Reconciler

var _ ReconcilerBuilder = NewReconciler

func NewReconciler(clt client.Client, apiReader client.Reader, dtc dtclient.Client, dk *dynakube.DynaKube) controllers.Reconciler {
	return &reconciler{
		client:       clt,
		apiReader:    apiReader,
		dk:           dk,
		dtc:          dtc,
		timeProvider: timeprovider.New(),
	}
}

var NoOneAgentCommunicationHostsError = errors.New("no communication hosts for OneAgent are available")

func (r *reconciler) Reconcile(ctx context.Context) error {
	if !r.dk.NeedAppInjection() && !r.dk.NeedsOneAgent() && !r.dk.LogMonitoring().IsEnabled() {
		if meta.FindStatusCondition(*r.dk.Conditions(), oaConnectionInfoConditionType) == nil {
			return nil // no condition == nothing is there to clean up
		}

		query := k8ssecret.Query(r.client, r.apiReader, log)
		err := query.Delete(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: r.dk.OneagentTenantSecret(), Namespace: r.dk.Namespace}})

		if err != nil {
			log.Error(err, "failed to clean-up OneAgent tenant-secret")
		}

		meta.RemoveStatusCondition(r.dk.Conditions(), oaConnectionInfoConditionType)
		r.dk.Status.OneAgent.ConnectionInfoStatus = dynakube.OneAgentConnectionInfoStatus{}

		return nil // clean-up shouldn't cause a failure
	}

	oldStatus := r.dk.Status.DeepCopy()

	err := r.reconcileConnectionInfo(ctx)
	if err != nil {
		return err
	}

	needStatusUpdate, err := hasher.IsDifferent(oldStatus, r.dk.Status)
	if err != nil {
		return errors.WithMessage(err, "failed to compare connection info status hashes")
	} else if needStatusUpdate {
		err = r.dk.UpdateStatus(ctx, r.client)
	}

	return err
}

func (r *reconciler) reconcileConnectionInfo(ctx context.Context) error {
	secretNamespacedName := types.NamespacedName{Name: r.dk.OneagentTenantSecret(), Namespace: r.dk.Namespace}

	if !conditions.IsOutdated(r.timeProvider, r.dk, oaConnectionInfoConditionType) {
		isSecretPresent, err := connectioninfo.IsTenantSecretPresent(ctx, r.apiReader, secretNamespacedName, log)
		if err != nil {
			return err
		}

		condition := meta.FindStatusCondition(*r.dk.Conditions(), oaConnectionInfoConditionType)
		if isSecretPresent {
			log.Info(dynakube.GetCacheValidMessage(
				"OneAgent connection info update",
				condition.LastTransitionTime,
				r.dk.ApiRequestThreshold()))

			return nil
		}
	}

	conditions.SetSecretOutdated(r.dk.Conditions(), oaConnectionInfoConditionType, secretNamespacedName.Name+" is not present or outdated, update in progress") // Necessary to update the LastTransitionTime, also it is a nice failsafe

	connectionInfo, err := r.dtc.GetOneAgentConnectionInfo(ctx)
	if err != nil {
		conditions.SetDynatraceApiError(r.dk.Conditions(), oaConnectionInfoConditionType, err)

		return errors.WithMessage(err, "failed to get OneAgent connection info")
	}

	r.setDynakubeStatus(connectionInfo)

	log.Info("OneAgent connection info updated")

	if len(connectionInfo.Endpoints) == 0 {
		log.Info("tenant has no endpoints", "tenant", connectionInfo.TenantUUID)
	}

	if len(connectionInfo.CommunicationHosts) == 0 {
		log.Info("no OneAgent communication hosts received, tenant API requests not yet throttled")
		setEmptyCommunicationHostsCondition(r.dk.Conditions())

		return NoOneAgentCommunicationHostsError
	}

	err = r.createTenantTokenSecret(ctx, r.dk.OneagentTenantSecret(), connectionInfo.ConnectionInfo)
	if err != nil {
		return err
	}

	log.Info("received OneAgent communication hosts", "communication hosts", connectionInfo.CommunicationHosts, "tenant", connectionInfo.TenantUUID)

	return nil
}

func (r *reconciler) setDynakubeStatus(connectionInfo dtclient.OneAgentConnectionInfo) {
	r.dk.Status.OneAgent.ConnectionInfoStatus.TenantUUID = connectionInfo.TenantUUID
	r.dk.Status.OneAgent.ConnectionInfoStatus.Endpoints = connectionInfo.Endpoints
	copyCommunicationHosts(&r.dk.Status.OneAgent.ConnectionInfoStatus, connectionInfo.CommunicationHosts)
}

func copyCommunicationHosts(dest *dynakube.OneAgentConnectionInfoStatus, src []dtclient.CommunicationHost) {
	dest.CommunicationHosts = make([]dynakube.CommunicationHostStatus, 0, len(src))
	for _, host := range src {
		dest.CommunicationHosts = append(dest.CommunicationHosts, dynakube.CommunicationHostStatus{
			Protocol: host.Protocol,
			Host:     host.Host,
			Port:     host.Port,
		})
	}
}

func (r *reconciler) createTenantTokenSecret(ctx context.Context, secretName string, connectionInfo dtclient.ConnectionInfo) error {
	secret, err := connectioninfo.BuildTenantSecret(r.dk, secretName, connectionInfo)
	if err != nil {
		return errors.WithStack(err)
	}

	query := k8ssecret.Query(r.client, r.apiReader, log)

	_, err = query.CreateOrUpdate(ctx, secret)
	if err != nil {
		log.Info("could not create or update secret for connection info", "name", secret.Name)
		conditions.SetKubeApiError(r.dk.Conditions(), oaConnectionInfoConditionType, err)

		return err
	}

	conditions.SetSecretCreated(r.dk.Conditions(), oaConnectionInfoConditionType, secret.Name)

	return nil
}

package oaconnectioninfo

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8ssecret"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type reconciler struct {
	client       client.Client
	dtc          dtclient.Client
	timeProvider *timeprovider.Provider
	dk           *dynakube.DynaKube
	secrets      k8ssecret.QueryObject
}
type ReconcilerBuilder func(clt client.Client, apiReader client.Reader, dtc dtclient.Client, dk *dynakube.DynaKube) controllers.Reconciler

var _ ReconcilerBuilder = NewReconciler

func NewReconciler(clt client.Client, apiReader client.Reader, dtc dtclient.Client, dk *dynakube.DynaKube) controllers.Reconciler {
	return &reconciler{
		client:       clt,
		dk:           dk,
		dtc:          dtc,
		timeProvider: timeprovider.New(),
		secrets:      k8ssecret.Query(clt, apiReader, log),
	}
}

var NoOneAgentCommunicationHostsError = errors.New("no communication hosts for OneAgent are available")

func (r *reconciler) Reconcile(ctx context.Context) error {
	if !r.dk.OneAgent().IsAppInjectionNeeded() && !r.dk.OneAgent().IsDaemonsetRequired() && !r.dk.LogMonitoring().IsEnabled() {
		if meta.FindStatusCondition(*r.dk.Conditions(), oaConnectionInfoConditionType) == nil {
			return nil // no condition == nothing is there to clean up
		}

		err := r.secrets.Delete(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: r.dk.OneAgent().GetTenantSecret(), Namespace: r.dk.Namespace}})
		if err != nil {
			log.Error(err, "failed to clean-up OneAgent tenant-secret")
		}

		meta.RemoveStatusCondition(r.dk.Conditions(), oaConnectionInfoConditionType)
		r.dk.Status.OneAgent.ConnectionInfoStatus = oneagent.ConnectionInfoStatus{}

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
	connectionInfo, err := r.dtc.GetOneAgentConnectionInfo(ctx)
	if err != nil {
		conditions.SetDynatraceAPIError(r.dk.Conditions(), oaConnectionInfoConditionType, err)

		return errors.WithMessage(err, "failed to get OneAgent connection info")
	}

	r.setDynakubeStatus(connectionInfo)

	log.Info("OneAgent connection info updated")

	if len(connectionInfo.Endpoints) == 0 {
		log.Info("no OneAgent communication endpoints received")
		setEmptyCommunicationHostsCondition(r.dk.Conditions())

		return NoOneAgentCommunicationHostsError
	}

	err = r.createTenantTokenSecret(ctx, r.dk.OneAgent().GetTenantSecret(), connectionInfo.ConnectionInfo)
	if err != nil {
		return err
	}

	r.dk.Status.OneAgent.ConnectionInfoStatus.TenantTokenHash, err = hasher.GenerateHash(connectionInfo.TenantToken)
	if err != nil {
		return errors.Wrap(err, "failed to generate TenantTokenHash")
	}

	log.Info("received OneAgent communication hosts", "communication hosts", connectionInfo.CommunicationHosts, "tenant", connectionInfo.TenantUUID)

	return nil
}

func (r *reconciler) setDynakubeStatus(connectionInfo dtclient.OneAgentConnectionInfo) {
	r.dk.Status.OneAgent.ConnectionInfoStatus.TenantUUID = connectionInfo.TenantUUID
	r.dk.Status.OneAgent.ConnectionInfoStatus.Endpoints = connectionInfo.Endpoints
	copyCommunicationHosts(&r.dk.Status.OneAgent.ConnectionInfoStatus, connectionInfo.CommunicationHosts)
}

func copyCommunicationHosts(dest *oneagent.ConnectionInfoStatus, src []dtclient.CommunicationHost) {
	dest.CommunicationHosts = make([]oneagent.CommunicationHostStatus, 0, len(src))
	for _, host := range src {
		dest.CommunicationHosts = append(dest.CommunicationHosts, oneagent.CommunicationHostStatus{
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

	_, err = r.secrets.CreateOrUpdate(ctx, secret)
	if err != nil {
		log.Info("could not create or update secret for connection info", "name", secret.Name)
		conditions.SetKubeAPIError(r.dk.Conditions(), oaConnectionInfoConditionType, err)

		return err
	}

	conditions.SetSecretCreated(r.dk.Conditions(), oaConnectionInfoConditionType, secret.Name)

	return nil
}

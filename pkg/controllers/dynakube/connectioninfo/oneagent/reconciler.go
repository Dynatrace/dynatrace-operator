package oaconnectioninfo

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8ssecret"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler struct {
	client       client.Client
	timeProvider *timeprovider.Provider
	secrets      k8ssecret.QueryObject
}

func NewReconciler(clt client.Client, apiReader client.Reader) *Reconciler {
	return &Reconciler{
		client:       clt,
		timeProvider: timeprovider.New(),
		secrets:      k8ssecret.Query(clt, apiReader, log),
	}
}

var NoOneAgentCommunicationEndpointsError = errors.New("no communication endpoints for OneAgent are available")

func (r *Reconciler) Reconcile(ctx context.Context, dk *dynakube.DynaKube, dtClient dtclient.Client) error {
	if !dk.OneAgent().IsAppInjectionNeeded() && !dk.OneAgent().IsDaemonsetRequired() && !dk.LogMonitoring().IsEnabled() {
		if meta.FindStatusCondition(*dk.Conditions(), oaConnectionInfoConditionType) == nil {
			return nil // no condition == nothing is there to clean up
		}

		err := r.secrets.Delete(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: dk.OneAgent().GetTenantSecret(), Namespace: dk.Namespace}})
		if err != nil {
			log.Error(err, "failed to clean-up OneAgent tenant-secret")
		}

		meta.RemoveStatusCondition(dk.Conditions(), oaConnectionInfoConditionType)
		dk.Status.OneAgent.ConnectionInfo = communication.ConnectionInfo{}

		return nil // clean-up shouldn't cause a failure
	}

	oldStatus := dk.Status.DeepCopy()

	err := r.reconcileConnectionInfo(ctx, dk, dtClient)
	if err != nil {
		return err
	}

	needStatusUpdate, err := hasher.IsDifferent(oldStatus, dk.Status)
	if err != nil {
		return errors.WithMessage(err, "failed to compare connection info status hashes")
	} else if needStatusUpdate {
		err = dk.UpdateStatus(ctx, r.client)
	}

	return err
}

func (r *Reconciler) reconcileConnectionInfo(ctx context.Context, dk *dynakube.DynaKube, dtc dtclient.Client) error {
	secretNamespacedName := types.NamespacedName{Name: dk.OneAgent().GetTenantSecret(), Namespace: dk.Namespace}

	if !k8sconditions.IsOutdated(r.timeProvider, dk, oaConnectionInfoConditionType) {
		isSecretPresent, err := connectioninfo.IsTenantSecretPresent(ctx, r.secrets, secretNamespacedName, log)
		if err != nil {
			return err
		}

		condition := meta.FindStatusCondition(*dk.Conditions(), oaConnectionInfoConditionType)
		if isSecretPresent {
			log.Info(dynakube.GetCacheValidMessage(
				"OneAgent connection info update",
				condition.LastTransitionTime,
				dk.APIRequestThreshold()))

			return nil
		}
	}

	k8sconditions.SetSecretOutdated(dk.Conditions(), oaConnectionInfoConditionType, secretNamespacedName.Name+" is not present or outdated, update in progress") // Necessary to update the LastTransitionTime, also it is a nice failsafe

	connectionInfo, err := dtc.GetOneAgentConnectionInfo(ctx)
	if err != nil {
		k8sconditions.SetDynatraceAPIError(dk.Conditions(), oaConnectionInfoConditionType, err)

		return errors.WithMessage(err, "failed to get OneAgent connection info")
	}

	r.setDynakubeStatus(connectionInfo, dk)

	log.Info("OneAgent connection info updated")

	if len(connectionInfo.Endpoints) == 0 {
		log.Info("no received OneAgent connection info, tenant API requests not yet throttled", "tenant", connectionInfo.TenantUUID)
		setEmptyCommunicationHostsCondition(dk.Conditions())

		return NoOneAgentCommunicationEndpointsError
	}

	err = r.createTenantTokenSecret(ctx, dk.OneAgent().GetTenantSecret(), connectionInfo.ConnectionInfo, dk)
	if err != nil {
		return err
	}

	dk.Status.OneAgent.ConnectionInfo.TenantTokenHash, err = hasher.GenerateHash(connectionInfo.TenantToken)
	if err != nil {
		return errors.Wrap(err, "failed to generate TenantTokenHash")
	}

	log.Info("received OneAgent connection info", "communication endpoints", connectionInfo.Endpoints, "tenant", connectionInfo.TenantUUID)

	return nil
}

func (r *Reconciler) setDynakubeStatus(connectionInfo dtclient.OneAgentConnectionInfo, dk *dynakube.DynaKube) {
	dk.Status.OneAgent.ConnectionInfo.TenantUUID = connectionInfo.TenantUUID
	dk.Status.OneAgent.ConnectionInfo.Endpoints = connectionInfo.Endpoints
}

func (r *Reconciler) createTenantTokenSecret(ctx context.Context, secretName string, connectionInfo dtclient.ConnectionInfo, dk *dynakube.DynaKube) error {
	secret, err := connectioninfo.BuildTenantSecret(dk, secretName, connectionInfo.TenantToken)
	if err != nil {
		return errors.WithStack(err)
	}

	_, err = r.secrets.CreateOrUpdate(ctx, secret)
	if err != nil {
		log.Info("could not create or update secret for connection info", "name", secret.Name)
		k8sconditions.SetKubeAPIError(dk.Conditions(), oaConnectionInfoConditionType, err)

		return err
	}

	k8sconditions.SetSecretCreated(dk.Conditions(), oaConnectionInfoConditionType, secret.Name)

	return nil
}

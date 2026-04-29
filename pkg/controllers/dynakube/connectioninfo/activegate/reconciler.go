package activegate

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
	agclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
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
	timeProvider *timeprovider.Provider
	secrets      k8ssecret.QueryObject
}

func NewReconciler(clt client.Client, apiReader client.Reader) *Reconciler {
	return &Reconciler{
		timeProvider: timeprovider.New(),
		secrets:      k8ssecret.Query(clt, apiReader, log),
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, agClient agclient.Client, dk *dynakube.DynaKube) error {
	if !dk.ActiveGate().IsEnabled() {
		if meta.FindStatusCondition(*dk.Conditions(), activeGateConnectionInfoConditionType) == nil {
			return nil
		}

		dk.Status.ActiveGate.ConnectionInfo = communication.ConnectionInfo{}

		err := r.secrets.Delete(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: dk.ActiveGate().GetTenantSecretName(), Namespace: dk.Namespace}})
		if err != nil {
			log.Error(err, "failed to clean-up ActiveGate tenant-secret")
		}

		meta.RemoveStatusCondition(dk.Conditions(), activeGateConnectionInfoConditionType)

		return nil // clean-up shouldn't cause a failure
	}

	err := r.reconcileConnectionInfo(ctx, dk, agClient)
	if err != nil {
		return err
	}

	return err
}

func (r *Reconciler) reconcileConnectionInfo(ctx context.Context, dk *dynakube.DynaKube, agClient agclient.Client) error {
	secretNamespacedName := types.NamespacedName{Name: dk.ActiveGate().GetTenantSecretName(), Namespace: dk.Namespace}

	if !k8sconditions.IsOutdated(r.timeProvider, dk, activeGateConnectionInfoConditionType) {
		isSecretPresent, err := connectioninfo.IsTenantSecretPresent(ctx, r.secrets, secretNamespacedName, log)
		if err != nil {
			return err
		}

		condition := meta.FindStatusCondition(*dk.Conditions(), activeGateConnectionInfoConditionType)
		if isSecretPresent {
			log.Info(dynakube.GetCacheValidMessage(
				"activegate connection info update",
				condition.LastTransitionTime,
				dk.APIRequestThreshold()))

			return nil
		}
	}

	k8sconditions.SetSecretOutdated(dk.Conditions(), activeGateConnectionInfoConditionType, secretNamespacedName.Name+" is not present or outdated, update in progress") // Necessary to update the LastTransitionTime, also it is a nice failsafe

	connectionInfo, err := agClient.GetConnectionInfo(ctx)
	if err != nil {
		k8sconditions.SetDynatraceAPIError(dk.Conditions(), activeGateConnectionInfoConditionType, err)

		return errors.WithMessage(err, "failed to get ActiveGate connection info")
	}

	r.setDynakubeStatus(dk, connectionInfo)

	if len(connectionInfo.Endpoints) == 0 {
		log.Info("tenant has no endpoints", "tenant", connectionInfo.TenantUUID)
	}

	err = r.createTenantTokenSecret(ctx, dk, dk.ActiveGate().GetTenantSecretName(), connectionInfo)
	if err != nil {
		return err
	}

	dk.Status.ActiveGate.ConnectionInfo.TenantTokenHash, err = hasher.GenerateHash(connectionInfo.TenantToken)
	if err != nil {
		return errors.Wrap(err, "failed to generate TenantTokenHash")
	}

	log.Info("activegate connection info updated")

	return nil
}

func (r *Reconciler) setDynakubeStatus(dk *dynakube.DynaKube, connectionInfo agclient.ConnectionInfo) {
	dk.Status.ActiveGate.ConnectionInfo.TenantUUID = connectionInfo.TenantUUID
	dk.Status.ActiveGate.ConnectionInfo.Endpoints = connectionInfo.Endpoints
}

func (r *Reconciler) createTenantTokenSecret(ctx context.Context, dk *dynakube.DynaKube, secretName string, connectionInfo agclient.ConnectionInfo) error {
	secret, err := connectioninfo.BuildTenantSecret(dk, k8slabel.ActiveGateComponentLabel, secretName, connectionInfo.TenantToken)
	if err != nil {
		return errors.WithStack(err)
	}

	_, err = r.secrets.CreateOrUpdate(ctx, secret)
	if err != nil {
		log.Info("could not create or update secret for connection info", "name", secret.Name)
		k8sconditions.SetKubeAPIError(dk.Conditions(), activeGateConnectionInfoConditionType, err)

		return err
	}

	k8sconditions.SetSecretCreated(dk.Conditions(), activeGateConnectionInfoConditionType, secret.Name)

	return nil
}

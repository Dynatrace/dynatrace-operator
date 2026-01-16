package activegate

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8ssecret"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type reconciler struct {
	dtc     dtclient.Client
	dk      *dynakube.DynaKube
	secrets k8ssecret.QueryObject
}

type ReconcilerBuilder func(clt client.Client, apiReader client.Reader, dtc dtclient.Client, dk *dynakube.DynaKube) controllers.Reconciler

var _ ReconcilerBuilder = NewReconciler

func NewReconciler(clt client.Client, apiReader client.Reader, dtc dtclient.Client, dk *dynakube.DynaKube) controllers.Reconciler {
	return &reconciler{
		dk:      dk,
		dtc:     dtc,
		secrets: k8ssecret.Query(clt, apiReader, log),
	}
}

func (r *reconciler) Reconcile(ctx context.Context) error {
	if !r.dk.ActiveGate().IsEnabled() {
		if meta.FindStatusCondition(*r.dk.Conditions(), activeGateConnectionInfoConditionType) == nil {
			return nil
		}

		r.dk.Status.ActiveGate.ConnectionInfo = communication.ConnectionInfo{}

		err := r.secrets.Delete(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: r.dk.ActiveGate().GetTenantSecretName(), Namespace: r.dk.Namespace}})
		if err != nil {
			log.Error(err, "failed to clean-up ActiveGate tenant-secret")
		}

		meta.RemoveStatusCondition(r.dk.Conditions(), activeGateConnectionInfoConditionType)

		return nil // clean-up shouldn't cause a failure
	}

	err := r.reconcileConnectionInfo(ctx)
	if err != nil {
		return err
	}

	return err
}

func (r *reconciler) reconcileConnectionInfo(ctx context.Context) error {
	secretNamespacedName := types.NamespacedName{Name: r.dk.ActiveGate().GetTenantSecretName(), Namespace: r.dk.Namespace}

	if r.dk.Status.DynatraceAPI.Throttled {
		isSecretPresent, err := connectioninfo.IsTenantSecretPresent(ctx, r.secrets, secretNamespacedName, log)
		if err != nil {
			return err
		}

		if isSecretPresent {
			log.Info(dynakube.GetCacheValidMessage(
				"activegate connection info update",
				r.dk.Status.DynatraceAPI.LastRequestPeriod,
				r.dk.APIRequestThreshold()))

			return nil
		}
	}

	connectionInfo, err := r.dtc.GetActiveGateConnectionInfo(ctx)
	if err != nil {
		k8sconditions.SetDynatraceAPIError(r.dk.Conditions(), activeGateConnectionInfoConditionType, err)

		return errors.WithMessage(err, "failed to get ActiveGate connection info")
	}

	r.setDynakubeStatus(connectionInfo)

	if len(connectionInfo.Endpoints) == 0 {
		log.Info("tenant has no endpoints", "tenant", connectionInfo.TenantUUID)
	}

	err = r.createTenantTokenSecret(ctx, r.dk.ActiveGate().GetTenantSecretName(), connectionInfo.ConnectionInfo)
	if err != nil {
		return err
	}

	r.dk.Status.ActiveGate.ConnectionInfo.TenantTokenHash, err = hasher.GenerateHash(connectionInfo.TenantToken)
	if err != nil {
		return errors.Wrap(err, "failed to generate TenantTokenHash")
	}

	log.Info("activegate connection info updated")

	return nil
}

func (r *reconciler) setDynakubeStatus(connectionInfo dtclient.ActiveGateConnectionInfo) {
	r.dk.Status.ActiveGate.ConnectionInfo.TenantUUID = connectionInfo.TenantUUID
	r.dk.Status.ActiveGate.ConnectionInfo.Endpoints = connectionInfo.Endpoints
}

func (r *reconciler) createTenantTokenSecret(ctx context.Context, secretName string, connectionInfo dtclient.ConnectionInfo) error {
	secret, err := connectioninfo.BuildTenantSecret(r.dk, secretName, connectionInfo)
	if err != nil {
		return errors.WithStack(err)
	}

	_, err = r.secrets.CreateOrUpdate(ctx, secret)
	if err != nil {
		log.Info("could not create or update secret for connection info", "name", secret.Name)
		k8sconditions.SetKubeAPIError(r.dk.Conditions(), activeGateConnectionInfoConditionType, err)

		return err
	}

	k8sconditions.SetSecretCreated(r.dk.Conditions(), activeGateConnectionInfoConditionType, secret.Name)

	return nil
}

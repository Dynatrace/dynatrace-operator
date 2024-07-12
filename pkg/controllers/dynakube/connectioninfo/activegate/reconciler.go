package activegate

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	k8ssecret "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/pkg/errors"
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

	dynakube *dynakube.DynaKube
}

type ReconcilerBuilder func(clt client.Client, apiReader client.Reader, dtc dtclient.Client, dk *dynakube.DynaKube) controllers.Reconciler

var _ ReconcilerBuilder = NewReconciler

func NewReconciler(clt client.Client, apiReader client.Reader, dtc dtclient.Client, dk *dynakube.DynaKube) controllers.Reconciler {
	return &reconciler{
		client:       clt,
		apiReader:    apiReader,
		dynakube:     dk,
		dtc:          dtc,
		timeProvider: timeprovider.New(),
	}
}

func (r *reconciler) Reconcile(ctx context.Context) error {
	if !r.dynakube.NeedsActiveGate() {
		if meta.FindStatusCondition(*r.dynakube.Conditions(), activeGateConnectionInfoConditionType) == nil {
			return nil
		}

		r.dynakube.Status.ActiveGate.ConnectionInfoStatus = dynakube.ActiveGateConnectionInfoStatus{}
		query := k8ssecret.NewQuery(ctx, r.client, r.apiReader, log)

		err := query.Delete(r.dynakube.ActivegateTenantSecret(), r.dynakube.Namespace)
		if err != nil {
			log.Error(err, "failed to clean-up ActiveGate tenant-secret")
		}

		meta.RemoveStatusCondition(r.dynakube.Conditions(), activeGateConnectionInfoConditionType)

		return nil // clean-up shouldn't cause a failure
	}

	err := r.reconcileConnectionInfo(ctx)

	if err != nil {
		return err
	}

	return err
}

func (r *reconciler) reconcileConnectionInfo(ctx context.Context) error {
	secretNamespacedName := types.NamespacedName{Name: r.dynakube.ActivegateTenantSecret(), Namespace: r.dynakube.Namespace}

	if !conditions.IsOutdated(r.timeProvider, r.dynakube, activeGateConnectionInfoConditionType) {
		isSecretPresent, err := connectioninfo.IsTenantSecretPresent(ctx, r.apiReader, secretNamespacedName, log)
		if err != nil {
			return err
		}

		condition := meta.FindStatusCondition(*r.dynakube.Conditions(), activeGateConnectionInfoConditionType)
		if isSecretPresent {
			log.Info(dynakube.GetCacheValidMessage(
				"activegate connection info update",
				condition.LastTransitionTime,
				r.dynakube.ApiRequestThreshold()))

			return nil
		}
	}

	conditions.SetSecretOutdated(r.dynakube.Conditions(), activeGateConnectionInfoConditionType, secretNamespacedName.Name+" is not present or outdated, update in progress") // Necessary to update the LastTransitionTime, also it is a nice failsafe

	connectionInfo, err := r.dtc.GetActiveGateConnectionInfo(ctx)
	if err != nil {
		conditions.SetDynatraceApiError(r.dynakube.Conditions(), activeGateConnectionInfoConditionType, err)

		return errors.WithMessage(err, "failed to get ActiveGate connection info")
	}

	r.setDynakubeStatus(connectionInfo)

	if len(connectionInfo.Endpoints) == 0 {
		log.Info("tenant has no endpoints", "tenant", connectionInfo.TenantUUID)
	}

	err = r.createTenantTokenSecret(ctx, r.dynakube.ActivegateTenantSecret(), r.dynakube, connectionInfo.ConnectionInfo)
	if err != nil {
		return err
	}

	log.Info("activegate connection info updated")

	return nil
}

func (r *reconciler) setDynakubeStatus(connectionInfo dtclient.ActiveGateConnectionInfo) {
	r.dynakube.Status.ActiveGate.ConnectionInfoStatus.TenantUUID = connectionInfo.TenantUUID
	r.dynakube.Status.ActiveGate.ConnectionInfoStatus.Endpoints = connectionInfo.Endpoints
}

func (r *reconciler) createTenantTokenSecret(ctx context.Context, secretName string, owner metav1.Object, connectionInfo dtclient.ConnectionInfo) error {
	secret, err := connectioninfo.BuildTenantSecret(owner, secretName, connectionInfo)
	if err != nil {
		return errors.WithStack(err)
	}

	query := k8ssecret.NewQuery(ctx, r.client, r.apiReader, log)

	err = query.CreateOrUpdate(*secret)
	if err != nil {
		log.Info("could not create or update secret for connection info", "name", secret.Name)
		conditions.SetKubeApiError(r.dynakube.Conditions(), activeGateConnectionInfoConditionType, err)

		return err
	}

	conditions.SetSecretCreated(r.dynakube.Conditions(), activeGateConnectionInfoConditionType, secret.Name)

	return nil
}

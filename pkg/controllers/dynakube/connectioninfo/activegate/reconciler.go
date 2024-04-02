package activegate

import (
	"context"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	k8ssecret "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/meta"
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

func (r *reconciler) Reconcile(ctx context.Context) error {
	if !r.dynakube.NeedsActiveGate() {
		meta.RemoveStatusCondition(&r.dynakube.Status.Conditions, activeGateConnectionInfoConditionType)

		return nil
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
			log.Info(dynatracev1beta1.GetCacheValidMessage(
				"activegate connection info update",
				condition.LastTransitionTime,
				r.dynakube.FeatureApiRequestThreshold()))

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
	secret, err := connectioninfo.BuildTenantSecret(owner, r.scheme, secretName, connectionInfo)
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

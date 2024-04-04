package activegate

import (
	"context"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	k8ssecret "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/pkg/errors"
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
		// TODO: Add check to only clean up if condition is not nil (condition is added in another change)
		r.dynakube.Status.ActiveGate.ConnectionInfoStatus = dynatracev1beta1.ActiveGateConnectionInfoStatus{}
		query := k8ssecret.NewQuery(ctx, r.client, r.apiReader, log)

		err := query.Delete(r.dynakube.ActivegateTenantSecret(), r.dynakube.Namespace)
		if err != nil {
			log.Error(err, "failed to clean-up ActiveGate tenant-secret")
		}

		return nil // clean-up shouldn't cause a failure
	}

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
	secretNamespacedName := types.NamespacedName{Name: r.dynakube.ActivegateTenantSecret(), Namespace: r.dynakube.Namespace}
	isOutdated := r.dynakube.IsActiveGateConnectionInfoUpdateAllowed(r.timeProvider)

	if !isOutdated {
		isSecretPresent, err := connectioninfo.IsTenantSecretPresent(ctx, r.apiReader, secretNamespacedName, log)
		if err != nil {
			return err
		}

		if isSecretPresent {
			log.Info(dynatracev1beta1.GetCacheValidMessage(
				"activegate connection info update",
				r.dynakube.Status.ActiveGate.ConnectionInfoStatus.LastRequest,
				r.dynakube.FeatureApiRequestThreshold()))

			return nil
		}
	}

	connectionInfo, err := r.dtc.GetActiveGateConnectionInfo(ctx)
	if err != nil {
		log.Info("failed to get activegate connection info")

		return err
	}

	r.setDynakubeStatus(connectionInfo)

	err = r.createTenantTokenSecret(ctx, r.dynakube.ActivegateTenantSecret(), r.dynakube, connectionInfo.ConnectionInfo)
	if err != nil {
		return err
	}

	log.Info("activegate connection info updated")

	r.dynakube.Status.ActiveGate.ConnectionInfoStatus.LastRequest = metav1.Now()

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

		return err
	}

	return nil
}

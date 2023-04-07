package connectioninfo

import (
	"context"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/timeprovider"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler struct {
	context      context.Context
	client       client.Client
	apiReader    client.Reader
	dtc          dtclient.Client
	dynakube     *dynatracev1beta1.DynaKube
	scheme       *runtime.Scheme
	timeProvider *timeprovider.Provider
}

func NewReconciler(ctx context.Context, clt client.Client, apiReader client.Reader, scheme *runtime.Scheme, dynakube *dynatracev1beta1.DynaKube, dtc dtclient.Client) *Reconciler { //nolint:revive // argument-limit doesn't apply to constructors
	return &Reconciler{
		context:      ctx,
		client:       clt,
		apiReader:    apiReader,
		dynakube:     dynakube,
		scheme:       scheme,
		dtc:          dtc,
		timeProvider: timeprovider.New(),
	}
}

func (r *Reconciler) Reconcile() error {
	if !r.dynakube.FeatureDisableActivegateRawImage() {
		err := r.reconcileActiveGateConnectionInfo()
		if err != nil {
			return err
		}
	}

	err := r.reconcileOneAgentConnectionInfo()
	if err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) createTenantTokenSecret(secretName string, connectionInfo dtclient.ConnectionInfo) error {
	secretData := extractSensitiveData(connectionInfo)
	secret, err := kubeobjects.CreateSecret(r.scheme, r.dynakube,
		kubeobjects.NewSecretNameModifier(secretName),
		kubeobjects.NewSecretNamespaceModifier(r.dynakube.Namespace),
		kubeobjects.NewSecretDataModifier(secretData))
	if err != nil {
		return errors.WithStack(err)
	}

	query := kubeobjects.NewSecretQuery(r.context, r.client, r.apiReader, log)
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

func extractPublicData(connectionInfo dtclient.ConnectionInfo) map[string]string {
	data := map[string]string{}

	if connectionInfo.TenantUUID != "" {
		data[TenantUUIDName] = connectionInfo.TenantUUID
	}
	if connectionInfo.Endpoints != "" {
		data[CommunicationEndpointsName] = connectionInfo.Endpoints
	}
	return data
}

package tenantinfo

import (
	"context"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler struct {
	context   context.Context
	client    client.Client
	apiReader client.Reader
	dtc       dtclient.Client
	dynakube  *dynatracev1beta1.DynaKube
}

var _ kubeobjects.Reconciler = &Reconciler{}

func NewReconciler(ctx context.Context, clt client.Client, apiReader client.Reader, dynakube *dynatracev1beta1.DynaKube, dtc dtclient.Client) *Reconciler {
	return &Reconciler{
		context:   ctx,
		client:    clt,
		apiReader: apiReader,
		dynakube:  dynakube,
		dtc:       dtc,
	}
}

func (r *Reconciler) Reconcile() (update bool, err error) {
	if !r.dynakube.FeatureDisableActivegateRawImage() {
		tenantInfo, err := r.dtc.GetActiveGateConnectionInfo()
		if err != nil {
			log.Info("failed to get activegate tenant info")
			return false, errors.WithStack(err)
		}

		data := buildTenantInfoSecret(tenantInfo.TenantToken, tenantInfo.TenantUUID, tenantInfo.Endpoints)
		secret := kubeobjects.NewSecret(r.dynakube.ActivegateTenantSecret(), r.dynakube.Namespace, data)
		err = r.createOrUpdateSecret(secret)
		if err != nil {
			return false, err
		}
	}

	if r.dynakube.FeatureOneAgentUseImmutableImage() {
		connectionInfo, err := r.dtc.GetOneAgentConnectionInfo()
		if err != nil {
			log.Info("failed to get oneagent connection info")
			return false, errors.WithStack(err)
		}

		data := buildTenantInfoSecret(
			connectionInfo.TenantToken, connectionInfo.TenantUUID, connectionInfo.Endpoints)
		secret := kubeobjects.NewSecret(r.dynakube.OneagentTenantSecret(), r.dynakube.Namespace, data)
		err = r.createOrUpdateSecret(secret)
		if err != nil {
			return false, err
		}
	}

	return false, nil
}

func buildTenantInfoSecret(token, uuid, endpoints string) map[string][]byte {
	// todo: interface for connectioninfo/tenantinfo
	data := map[string][]byte{
		TenantTokenName: []byte(token),
	}
	if uuid != "" {
		data[TenantUuidName] = []byte(uuid)
	}
	if endpoints != "" {
		data[CommunicationEndpointsName] = []byte(endpoints)
	}

	return data
}

func (r *Reconciler) createOrUpdateSecret(secret *corev1.Secret) error {
	query := kubeobjects.NewSecretQuery(r.context, r.client, r.apiReader, log)
	if err := query.CreateOrUpdate(*secret); err != nil {
		log.Info("could not create or update secret for tenant info", "name", secret.Name)
		return err
	}
	return nil
}

package connectioninfo

import (
	"context"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler struct {
	context   context.Context
	client    client.Client
	apiReader client.Reader
	dtc       dtclient.Client
	dynakube  *dynatracev1beta1.DynaKube
}

func NewReconciler(ctx context.Context, clt client.Client, apiReader client.Reader, dynakube *dynatracev1beta1.DynaKube, dtc dtclient.Client) *Reconciler {
	return &Reconciler{
		context:   ctx,
		client:    clt,
		apiReader: apiReader,
		dynakube:  dynakube,
		dtc:       dtc,
	}
}

func (r *Reconciler) Reconcile() (err error) {
	if !r.dynakube.FeatureDisableActivegateRawImage() {
		activeGateConnectionInfo, err := r.dtc.GetActiveGateConnectionInfo()
		if err != nil {
			log.Info("failed to get activegate connection info")
			return errors.WithStack(err)
		}

		err = r.createOrUpdateSecret(r.dynakube.ActivegateTenantSecret(), activeGateConnectionInfo.ConnectionInfo)
		if err != nil {
			return err
		}
	}

	if r.dynakube.FeatureOneAgentImmutableImage() {
		oneAgentConnectionInfo, err := r.dtc.GetOneAgentConnectionInfo()
		if err != nil {
			log.Info("failed to get oneagent connection info")
			return errors.WithStack(err)
		}

		err = r.createOrUpdateSecret(r.dynakube.OneagentTenantSecret(), oneAgentConnectionInfo.ConnectionInfo)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Reconciler) createOrUpdateSecret(secretName string, connectionInfo dtclient.ConnectionInfo) error {
	data := buildConnectionInfoSecret(connectionInfo)
	secret := kubeobjects.NewSecret(secretName, r.dynakube.Namespace, data)

	query := kubeobjects.NewSecretQuery(r.context, r.client, r.apiReader, log)
	err := query.CreateOrUpdate(*secret)
	if err != nil {
		log.Info("could not create or update secret for connection info", "name", secret.Name)
		return err
	}
	return nil
}

func buildConnectionInfoSecret(connectionInfo dtclient.ConnectionInfo) map[string][]byte {
	data := map[string][]byte{
		TenantTokenName: []byte(connectionInfo.TenantToken),
	}

	if connectionInfo.TenantUUID != "" {
		data[TenantUuidName] = []byte(connectionInfo.TenantUUID)
	}
	if connectionInfo.Endpoints != "" {
		data[CommunicationEndpointsName] = []byte(connectionInfo.Endpoints)
	}

	return data
}

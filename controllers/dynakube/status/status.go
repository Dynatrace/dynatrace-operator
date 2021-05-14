package status

import (
	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/kubesystem"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Options struct {
	dtc       dtclient.Client
	apiClient client.Client
}

func SetDynakubeStatus(instance *dynatracev1alpha1.DynaKube, opts Options) error {
	clt := opts.apiClient
	dtc := opts.dtc

	uid, err := kubesystem.GetUID(clt)
	if err != nil {
		return errors.WithStack(err)
	}

	communicationHost, err := dtc.GetCommunicationHostForClient()
	if err != nil {
		return errors.WithStack(err)
	}

	connectionInfo, err := dtc.GetConnectionInfo()
	if err != nil {
		return errors.WithStack(err)
	}

	instance.Status.KubeSystemUUID = string(uid)
	instance.Status.CommunicationHostForClient = communicationHost
	instance.Status.ConnectionInfo = connectionInfo
	instance.Status.EnvironmentID = connectionInfo.TenantUUID

	return nil
}

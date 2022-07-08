package status

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/kubesystem"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Options struct {
	Dtc       dtclient.Client
	ApiClient client.Reader
}

func SetDynakubeStatus(instance *dynatracev1beta1.DynaKube, opts Options) error {
	clt := opts.ApiClient
	dtc := opts.Dtc

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

	latestAgentVersionUnixDefault, err := dtc.GetLatestAgentVersion(
		dtclient.OsUnix, dtclient.InstallerTypeDefault)
	if err != nil {
		return errors.WithStack(err)
	}

	latestAgentVersionUnixPaas, err := dtc.GetLatestAgentVersion(
		dtclient.OsUnix, dtclient.InstallerTypePaaS)
	if err != nil {
		return errors.WithStack(err)
	}

	communicationHostStatus := dynatracev1beta1.CommunicationHostStatus(communicationHost)

	connectionInfoStatus := dynatracev1beta1.ConnectionInfoStatus{
		CommunicationHosts: communicationHostsToStatus(connectionInfo.CommunicationHosts),
		TenantUUID:         connectionInfo.TenantUUID,
	}

	instance.Status.KubeSystemUUID = string(uid)
	instance.Status.CommunicationHostForClient = communicationHostStatus
	instance.Status.ConnectionInfo = connectionInfoStatus
	instance.Status.LatestAgentVersionUnixDefault = latestAgentVersionUnixDefault
	instance.Status.LatestAgentVersionUnixPaas = latestAgentVersionUnixPaas

	return nil
}

func communicationHostsToStatus(communicationHosts []dtclient.CommunicationHost) []dynatracev1beta1.CommunicationHostStatus {
	var communicationHostStatuses []dynatracev1beta1.CommunicationHostStatus

	for _, communicationHost := range communicationHosts {
		communicationHostStatuses = append(communicationHostStatuses, dynatracev1beta1.CommunicationHostStatus(communicationHost))
	}

	return communicationHostStatuses
}

package status

import (
	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/kubesystem"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Options struct {
	Dtc       dtclient.Client
	ApiClient client.Reader
}

func SetDynakubeStatus(instance *dynatracev1alpha1.DynaKube, opts Options) error {
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

	tenantInfo, err := dtc.GetAgentTenantInfo()
	connectionInfo := tenantInfo.ConnectionInfo

	if err != nil {
		return errors.WithStack(err)
	}

	latestAgentVersionUnixDefault, err := dtc.GetLatestAgentVersion(dtclient.OsUnix, dtclient.InstallerTypeDefault)
	if err != nil {
		return errors.WithStack(err)
	}

	latestAgentVersionUnixPaas, err := dtc.GetLatestAgentVersion(dtclient.OsUnix, dtclient.InstallerTypePaaS)
	if err != nil {
		return errors.WithStack(err)
	}

	communicationHostStatus := dynatracev1alpha1.CommunicationHostStatus(*communicationHost)

	connectionInfoStatus := dynatracev1alpha1.ConnectionInfoStatus{
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

func communicationHostsToStatus(communicationHosts []*dtclient.CommunicationHost) []dynatracev1alpha1.CommunicationHostStatus {
	var communicationHostStatuses []dynatracev1alpha1.CommunicationHostStatus

	for _, communicationHost := range communicationHosts {
		communicationHostStatuses = append(communicationHostStatuses, dynatracev1alpha1.CommunicationHostStatus(*communicationHost))
	}

	return communicationHostStatuses
}

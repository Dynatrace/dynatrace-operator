package status

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/kubesystem"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Options struct {
	DtClient  dtclient.Client
	ApiReader client.Reader
}

func SetDynakubeStatus(dynakube *dynatracev1beta1.DynaKube, opts Options) error {
	apiReader := opts.ApiReader
	dtClient := opts.DtClient

	uid, err := kubesystem.GetUID(apiReader)
	if err != nil {
		log.Info("could not get cluster ID")
		return err
	}

	communicationHost, err := dtClient.GetCommunicationHostForClient()
	if err != nil {
		log.Info("could not get communication hosts")
		return err
	}

	connectionInfo, err := dtClient.GetOneAgentConnectionInfo()
	if err != nil {
		log.Info("could not get connection info")
		return err
	}

	latestAgentVersionUnixDefault, err := dtClient.GetLatestAgentVersion(
		dtclient.OsUnix, dtclient.InstallerTypeDefault)
	if err != nil {
		log.Info("could not get agent default unix version")
		return err
	}

	latestAgentVersionUnixPaas, err := dtClient.GetLatestAgentVersion(
		dtclient.OsUnix, dtclient.InstallerTypePaaS)
	if err != nil {
		log.Info("could not get agent paas unix version")
		return err
	}

	communicationHostStatus := dynatracev1beta1.CommunicationHostStatus(communicationHost)

	connectionInfoStatus := dynatracev1beta1.ConnectionInfoStatus{
		CommunicationHosts:              communicationHostsToStatus(connectionInfo.CommunicationHosts),
		TenantUUID:                      connectionInfo.TenantUUID,
		FormattedCommunicationEndpoints: connectionInfo.Endpoints,
	}

	dynakube.Status.KubeSystemUUID = string(uid)
	dynakube.Status.CommunicationHostForClient = communicationHostStatus
	dynakube.Status.ConnectionInfo = connectionInfoStatus
	dynakube.Status.LatestAgentVersionUnixDefault = latestAgentVersionUnixDefault
	dynakube.Status.LatestAgentVersionUnixPaas = latestAgentVersionUnixPaas
	dynakube.Status.Tokens = dynakube.Tokens()

	return nil
}

func communicationHostsToStatus(communicationHosts []dtclient.CommunicationHost) []dynatracev1beta1.CommunicationHostStatus {
	communicationHostStatuses := make([]dynatracev1beta1.CommunicationHostStatus, 0, len(communicationHosts))

	for _, communicationHost := range communicationHosts {
		communicationHostStatuses = append(communicationHostStatuses, dynatracev1beta1.CommunicationHostStatus(communicationHost))
	}

	return communicationHostStatuses
}

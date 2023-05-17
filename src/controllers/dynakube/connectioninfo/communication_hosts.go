package connectioninfo

import (
	dynatracev1 "github.com/Dynatrace/dynatrace-operator/src/api/v1"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
)

func GetCommunicationHosts(dynakube *dynatracev1.DynaKube) []dtclient.CommunicationHost {
	communicationHosts := make([]dtclient.CommunicationHost, 0, len(dynakube.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts))
	for _, host := range dynakube.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts {
		communicationHosts = append(communicationHosts, dtclient.CommunicationHost{
			Protocol: host.Protocol,
			Host:     host.Host,
			Port:     host.Port,
		})
	}
	return communicationHosts
}

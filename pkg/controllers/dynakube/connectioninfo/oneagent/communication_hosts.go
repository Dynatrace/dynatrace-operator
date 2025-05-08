package oaconnectioninfo

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta5/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
)

func GetCommunicationHosts(dk *dynakube.DynaKube) []dtclient.CommunicationHost {
	communicationHosts := make([]dtclient.CommunicationHost, 0, len(dk.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts))
	for _, host := range dk.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts {
		communicationHosts = append(communicationHosts, dtclient.CommunicationHost{
			Protocol: host.Protocol,
			Host:     host.Host,
			Port:     host.Port,
		})
	}

	return communicationHosts
}

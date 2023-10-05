package connectioninfo

import (
	"net"
	"net/url"
	"strconv"
	"strings"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/pkg/errors"
)

func GetOneAgentCommunicationHosts(dynakube *dynatracev1beta1.DynaKube) []dtclient.CommunicationHost {
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

func GetActiveGateEndpointsAsCommunicationHosts(dynakube *dynatracev1beta1.DynaKube) []dtclient.CommunicationHost {
	activegateEndpointsString := dynakube.Status.ActiveGate.ConnectionInfoStatus.Endpoints
	if activegateEndpointsString == "" {
		return []dtclient.CommunicationHost{}
	}
	return parseCommunicationHostFromActiveGateEndpoints(activegateEndpointsString)
}

func parseCommunicationHostFromActiveGateEndpoints(activegateEndpointsString string) []dtclient.CommunicationHost {
	endpointStrings := strings.Split(activegateEndpointsString, ",")
	endpointCommunicationHosts := make(map[dtclient.CommunicationHost]bool, len(endpointStrings))
	for _, endpointString := range endpointStrings {
		if endpoint, err := parseEndpointToCommunicationHost(endpointString); err == nil {
			endpointCommunicationHosts[endpoint] = true
		}
	}

	comHosts := make([]dtclient.CommunicationHost, 0, len(endpointCommunicationHosts))
	for ch := range endpointCommunicationHosts {
		comHosts = append(comHosts, ch)
	}

	return comHosts
}

func parseEndpointToCommunicationHost(endpointString string) (dtclient.CommunicationHost, error) {
	if endpointString == "" {
		return dtclient.CommunicationHost{}, errors.New("empty endpoint string not allowed")
	}
	parsedEndpoint, err := url.Parse(endpointString)

	if err != nil {
		return dtclient.CommunicationHost{}, err
	}
	host := parsedEndpoint.Host
	hostPart, portPart, err := net.SplitHostPort(host)

	if err != nil {
		return dtclient.CommunicationHost{}, err
	}

	intPort, err := strconv.ParseUint(portPart, 10, 32)
	if err != nil {
		return dtclient.CommunicationHost{}, err
	}

	return dtclient.CommunicationHost{
		Protocol: parsedEndpoint.Scheme,
		Host:     hostPart,
		Port:     uint32(intPort),
	}, nil
}

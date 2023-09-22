package connectioninfo

import (
	"net/url"
	"strconv"
	"strings"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/pkg/errors"
)

const (
	ActivegateEndpointPrefix = "endpoints: "
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
	if activegateEndpointsString == "" || !strings.HasPrefix(activegateEndpointsString, ActivegateEndpointPrefix) {
		return []dtclient.CommunicationHost{}
	}
	return parseCommunicationHostFromActiveGateEndpoints(activegateEndpointsString[len(ActivegateEndpointPrefix):])
}

func parseCommunicationHostFromActiveGateEndpoints(activegateEndpointsString string) []dtclient.CommunicationHost {
	endpointStrings := strings.Split(activegateEndpointsString, ";")
	endpointCommunicationHosts := make([]dtclient.CommunicationHost, 0)
	for _, endpointString := range endpointStrings {
		endpoint, err := parseEndpointToCommunicationHost(endpointString)
		if err == nil {
			endpointCommunicationHosts = append(endpointCommunicationHosts, endpoint)
		}
	}

	return endpointCommunicationHosts
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
	parts := strings.Split(host, ":")
	var port uint32

	if len(parts) > 1 {
		intPort, err := strconv.ParseUint(parts[1], 10, 32)
		if err != nil {
			return dtclient.CommunicationHost{}, err
		}
		port = uint32(intPort)
	} else {
		port = uint32(80)
	}
	return dtclient.CommunicationHost{
		Protocol: parsedEndpoint.Scheme,
		Host:     parts[0],
		Port:     port,
	}, nil
}

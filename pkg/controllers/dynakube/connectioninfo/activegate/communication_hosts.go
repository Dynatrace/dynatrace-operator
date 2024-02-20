package activegate

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/pkg/errors"
)

const (
	DefaultHttpPort = uint32(80)
)

func GetEndpointsAsCommunicationHosts(dynakube *dynakube.DynaKube) []dtclient.CommunicationHost {
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

	port, err := getPortOrDefault(parsedEndpoint, DefaultHttpPort)
	if err != nil {
		return dtclient.CommunicationHost{}, err
	}

	return dtclient.CommunicationHost{
		Protocol: parsedEndpoint.Scheme,
		Host:     parsedEndpoint.Hostname(),
		Port:     port,
	}, nil
}

func getPortOrDefault(u *url.URL, defaultPort uint32) (uint32, error) {
	portString := u.Port()

	if portString == "" {
		return defaultPort, nil
	}

	if p, err := strconv.ParseUint(portString, 10, 32); err == nil {
		return uint32(p), nil
	} else {
		return 0, err
	}
}

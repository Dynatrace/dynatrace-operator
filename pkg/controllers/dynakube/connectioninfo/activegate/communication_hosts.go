package activegate

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/pkg/errors"
)

const (
	DefaultHttpPort = uint32(80)
)

func GetEndpointsAsCommunicationHosts(dk *dynakube.DynaKube) []dtclient.CommunicationHost {
	activegateEndpointsString := dk.Status.ActiveGate.ConnectionInfo.Endpoints
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

	var p uint32

	_, err := fmt.Sscan(portString, &p)
	if err == nil {
		return p, nil
	} else {
		return 0, err
	}
}

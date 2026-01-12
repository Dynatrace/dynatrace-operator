package istio

import (
	"errors"
	"fmt"
	"maps"
	"net/url"
	"slices"
	"sort"
	"strings"
)

// CommunicationHost => struct of connection endpoint
type CommunicationHost struct {
	Protocol string
	Host     string
	Port     uint32
}

func (ch CommunicationHost) String() string {
	return fmt.Sprintf("%s://%s:%d", ch.Protocol, ch.Host, ch.Port)
}

func NewCommunicationHost(endpoint string) (CommunicationHost, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return CommunicationHost{}, errors.New("failed to parse URL")
	}

	if u.Scheme == "" {
		return CommunicationHost{}, errors.New("no protocol provided")
	} else if u.Scheme != "http" && u.Scheme != "https" {
		return CommunicationHost{}, errors.New("unknown protocol")
	}

	rp := u.Port() // Empty if not included in the URI

	var p uint32

	if rp == "" {
		switch u.Scheme {
		case "http":
			p = 80
		case "https":
			p = 443
		}
	} else {
		_, err := fmt.Sscan(rp, &p)
		if err != nil {
			return CommunicationHost{}, errors.New("failed to parse port")
		}
	}

	return CommunicationHost{
		Protocol: u.Scheme,
		Host:     u.Hostname(),
		Port:     p,
	}, nil
}

// NewCommunicationHosts creates CommunicationHost slice from comma separated endpoints.
// It removes duplicates and sorts the result for deterministic output.
func NewCommunicationHosts(endpoints string) ([]CommunicationHost, error) {
	// we want to avoid duplicates, because they cause problems with the istio "integration", as you should not have duplicate hosts in the ServiceEntries
	comHosts := map[string]CommunicationHost{}

	if len(endpoints) == 0 {
		return []CommunicationHost{}, nil
	}

	for _, endpoint := range strings.Split(endpoints, ",") {
		ch, err := NewCommunicationHost(endpoint)
		if err != nil {
			return nil, err
		}

		comHosts[ch.String()] = ch
	}

	// we have to sort the hosts to have a deterministic order for not constantly change the Status
	sortedHosts := slices.Collect(maps.Values(comHosts))
	sort.SliceStable(sortedHosts, func(i, j int) bool {
		return sortedHosts[i].String() < sortedHosts[j].String()
	})

	return sortedHosts, nil
}

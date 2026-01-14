package istio

import (
	"fmt"
	"maps"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/pkg/errors"
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
		return CommunicationHost{}, errors.WithMessage(err, "failed to parse URL")
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
		p64, err := strconv.ParseUint(rp, 10, 32)
		if err != nil {
			return CommunicationHost{}, errors.WithMessage(err, "failed to parse port")
		}

		p = uint32(p64)
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

	for endpoint := range strings.SplitSeq(endpoints, ",") {
		ch, err := NewCommunicationHost(endpoint)
		if err != nil {
			return nil, err
		}

		comHosts[ch.String()] = ch
	}

	// we have to sort the hosts to have a deterministic order for easier testing.

	sortedHosts := slices.SortedFunc(maps.Values(comHosts), func(a, b CommunicationHost) int {
		return strings.Compare(a.String(), b.String())
	})

	return sortedHosts, nil
}

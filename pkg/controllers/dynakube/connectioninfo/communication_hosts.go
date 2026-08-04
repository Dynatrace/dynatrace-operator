// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package connectioninfo

import (
	"fmt"
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

// ParseCommunicationHosts creates CommunicationHost slice from `,` separated endpoints.
// It sorts the result for deterministic output.
func ParseCommunicationHosts(endpoints string) ([]CommunicationHost, error) {
	return newCommunicationHosts(endpoints, ",")
}

func newCommunicationHosts(endpoints string, sep string) ([]CommunicationHost, error) {
	if len(endpoints) == 0 {
		return []CommunicationHost{}, nil
	}

	var comHosts []CommunicationHost

	for endpoint := range strings.SplitSeq(endpoints, sep) {
		ch, err := NewCommunicationHost(endpoint)
		if err != nil {
			return nil, err
		}

		comHosts = append(comHosts, ch)
	}

	// sort for deterministic output
	slices.SortFunc(comHosts, func(a, b CommunicationHost) int {
		return strings.Compare(a.String(), b.String())
	})

	return comHosts, nil
}

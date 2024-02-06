package dynatrace

import (
	"errors"
	"net/url"
	"strconv"
)

func (dtc *dynatraceClient) GetCommunicationHostForClient() (CommunicationHost, error) {
	return ParseEndpoint(dtc.url)
}

func ParseEndpoint(s string) (CommunicationHost, error) {
	u, err := url.ParseRequestURI(s)
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
		q, err := strconv.ParseUint(rp, 10, 32)
		if err != nil {
			return CommunicationHost{}, errors.New("failed to parse port")
		}

		p = uint32(q)
	}

	return CommunicationHost{
		Protocol: u.Scheme,
		Host:     u.Hostname(),
		Port:     p,
	}, nil
}

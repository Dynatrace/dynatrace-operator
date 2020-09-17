package dtclient

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
)

// CommunicationHost => struct of connection endpoint
type CommunicationHost struct {
	Protocol string
	Host     string
	Port     uint32
}

func (dc *dynatraceClient) GetCommunicationHostForClient() (CommunicationHost, error) {
	return dc.parseEndpoint(dc.url)
}

func (dc *dynatraceClient) GetCommunicationHosts() ([]CommunicationHost, error) {
	connectionInfoUrl := fmt.Sprintf("%s/v1/deployment/installer/agent/connectioninfo", dc.url)
	resp, err := dc.makeRequest(connectionInfoUrl, dynatracePaaSToken)
	if err != nil {
		return nil, err
	}
	defer func() {
		//Swallow error, nothing has to be done at this point
		_ = resp.Body.Close()
	}()

	responseData, err := dc.getServerResponseData(resp)
	if err != nil {
		return nil, err
	}

	return dc.readResponseForConnectionInfo(responseData)
}

func (dc *dynatraceClient) readResponseForConnectionInfo(response []byte) ([]CommunicationHost, error) {
	type jsonResponse struct {
		CommunicationEndpoints []string
	}

	resp := jsonResponse{}
	err := json.Unmarshal(response, &resp)
	if err != nil {
		logger.Error(err, "error unmarshalling json response")
		return nil, err
	}

	out := make([]CommunicationHost, 0, len(resp.CommunicationEndpoints))

	for _, s := range resp.CommunicationEndpoints {
		logger := logger.WithValues("url", s)

		e, err := dc.parseEndpoint(s)
		if err != nil {
			logger.Info("failed to parse communication endpoint")
			continue
		}
		out = append(out, e)
	}

	if len(out) == 0 {
		return nil, errors.New("no hosts available")
	}

	return out, nil
}

func (dc *dynatraceClient) parseEndpoint(s string) (CommunicationHost, error) {
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

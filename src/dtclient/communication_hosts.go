package dtclient

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
)

// ConnectionInfo => struct of TenantUUID and CommunicationHosts
type ConnectionInfo struct {
	CommunicationHosts []CommunicationHost
	TenantUUID         string
}

// CommunicationHost => struct of connection endpoint
type CommunicationHost struct {
	Protocol string
	Host     string
	Port     uint32
}

func (dtc *dynatraceClient) GetCommunicationHostForClient() (CommunicationHost, error) {
	return dtc.parseEndpoint(dtc.url)
}

func (dtc *dynatraceClient) GetConnectionInfo() (ConnectionInfo, error) {
	connectionInfoURL := fmt.Sprintf("%s/v1/deployment/installer/agent/connectioninfo", dtc.url)
	resp, err := dtc.makeRequest(connectionInfoURL, dynatracePaaSToken)
	if err != nil {
		return ConnectionInfo{}, err
	}
	defer func() {
		//Swallow error, nothing has to be done at this point
		_ = resp.Body.Close()
	}()

	responseData, err := dtc.getServerResponseData(resp)
	if err != nil {
		return ConnectionInfo{}, err
	}

	return dtc.readResponseForConnectionInfo(responseData)
}

func (dtc *dynatraceClient) readResponseForConnectionInfo(response []byte) (ConnectionInfo, error) {
	type jsonResponse struct {
		TenantUUID             string   `json:"tenantUUID"`
		CommunicationEndpoints []string `json:"communicationEndpoints"`
	}

	resp := jsonResponse{}
	err := json.Unmarshal(response, &resp)
	if err != nil {
		dtc.logger.Error(err, "error unmarshalling json response")
		return ConnectionInfo{}, err
	}

	t := resp.TenantUUID
	ch := make([]CommunicationHost, 0, len(resp.CommunicationEndpoints))

	for _, s := range resp.CommunicationEndpoints {
		logger := dtc.logger.WithValues("url", s)

		e, err := dtc.parseEndpoint(s)
		if err != nil {
			logger.Info("failed to parse communication endpoint")
			continue
		}
		ch = append(ch, e)
	}

	if len(ch) == 0 {
		return ConnectionInfo{}, errors.New("no communication hosts available")
	}

	ci := ConnectionInfo{
		CommunicationHosts: ch,
		TenantUUID:         t,
	}

	return ci, nil
}

func (dtc *dynatraceClient) parseEndpoint(s string) (CommunicationHost, error) {
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

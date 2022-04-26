package dtclient

import (
	"encoding/json"
	"errors"
	"net/url"
	"strconv"

	"github.com/Dynatrace/dynatrace-operator/src/dtclient/types"
)

func (dtc *dynatraceClient) GetCommunicationHostForClient() (types.CommunicationHost, error) {
	return dtc.parseEndpoint(dtc.url)
}

func (dtc *dynatraceClient) GetConnectionInfo() (types.ConnectionInfo, error) {
	resp, err := dtc.makeRequest(dtc.getOneAgentConnectionInfoUrl(), dynatracePaaSToken)
	if err != nil {
		return types.ConnectionInfo{}, err
	}
	defer func() {
		//Swallow error, nothing has to be done at this point
		_ = resp.Body.Close()
	}()

	responseData, err := dtc.getServerResponseData(resp)
	if err != nil {
		return types.ConnectionInfo{}, err
	}

	return dtc.readResponseForConnectionInfo(responseData)
}

func (dtc *dynatraceClient) readResponseForConnectionInfo(response []byte) (types.ConnectionInfo, error) {
	type jsonResponse struct {
		TenantUUID                      string   `json:"tenantUUID"`
		TenantToken                     string   `json:"tenantToken"`
		CommunicationEndpoints          []string `json:"communicationEndpoints"`
		FormattedCommunicationEndpoints string   `json:"formattedCommunicationEndpoints"`
	}

	resp := jsonResponse{}
	err := json.Unmarshal(response, &resp)
	if err != nil {
		log.Error(err, "error unmarshalling connection info response", "response", string(response))
		return types.ConnectionInfo{}, err
	}

	tenantUuid := resp.TenantUUID
	tenantToken := resp.TenantToken
	communicationHosts := make([]types.CommunicationHost, 0, len(resp.CommunicationEndpoints))
	formattedCommunicationEndpoints := resp.FormattedCommunicationEndpoints

	for _, s := range resp.CommunicationEndpoints {
		e, err := dtc.parseEndpoint(s)
		if err != nil {
			log.Info("failed to parse communication endpoint", "url", s)
			continue
		}
		communicationHosts = append(communicationHosts, e)
	}

	if len(communicationHosts) == 0 {
		return types.ConnectionInfo{}, errors.New("no communication hosts available")
	}

	ci := types.ConnectionInfo{
		CommunicationHosts:              communicationHosts,
		TenantUUID:                      tenantUuid,
		TenantToken:                     tenantToken,
		FormattedCommunicationEndpoints: formattedCommunicationEndpoints,
	}

	return ci, nil
}

func (dtc *dynatraceClient) parseEndpoint(s string) (types.CommunicationHost, error) {
	u, err := url.ParseRequestURI(s)
	if err != nil {
		return types.CommunicationHost{}, errors.New("failed to parse URL")
	}

	if u.Scheme == "" {
		return types.CommunicationHost{}, errors.New("no protocol provided")
	} else if u.Scheme != "http" && u.Scheme != "https" {
		return types.CommunicationHost{}, errors.New("unknown protocol")
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
			return types.CommunicationHost{}, errors.New("failed to parse port")
		}
		p = uint32(q)
	}

	return types.CommunicationHost{
		Protocol: u.Scheme,
		Host:     u.Hostname(),
		Port:     p,
	}, nil
}

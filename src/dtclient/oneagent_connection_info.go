package dtclient

import "encoding/json"

type OneAgentConnectionInfo struct {
	ConnectionInfo
	CommunicationHosts []CommunicationHost
}

type oneAgentConnectionInfoJsonResponse struct {
	TenantUUID                      string   `json:"tenantUUID"`
	TenantToken                     string   `json:"tenantToken"`
	CommunicationEndpoints          []string `json:"communicationEndpoints"`
	FormattedCommunicationEndpoints string   `json:"formattedCommunicationEndpoints"`
}

func (dtc *dynatraceClient) GetOneAgentConnectionInfo() (OneAgentConnectionInfo, error) {
	resp, err := dtc.makeRequest(dtc.getOneAgentConnectionInfoUrl(), dynatracePaaSToken)
	if err != nil {
		return OneAgentConnectionInfo{}, err
	}
	defer CloseBodyAfterRequest(resp)

	if resp.StatusCode == 400 {
		log.Info("server could not find the network zone or deliver default fallback config, is there an ActiveGate configured for the network zone?")
		return OneAgentConnectionInfo{}, nil
	}

	responseData, err := dtc.getServerResponseData(resp)
	if err != nil {
		return OneAgentConnectionInfo{}, dtc.handleErrorResponseFromAPI(responseData, resp.StatusCode)
	}

	connectionInfo, err := dtc.readResponseForOneAgentConnectionInfo(responseData)
	if err != nil {
		return OneAgentConnectionInfo{}, err
	}

	return connectionInfo, nil
}

func (dtc *dynatraceClient) readResponseForOneAgentConnectionInfo(response []byte) (OneAgentConnectionInfo, error) {
	resp := oneAgentConnectionInfoJsonResponse{}
	err := json.Unmarshal(response, &resp)
	if err != nil {
		log.Error(err, "error unmarshalling connection info response", "response", string(response))
		return OneAgentConnectionInfo{}, err
	}

	tenantUUID := resp.TenantUUID
	tenantToken := resp.TenantToken
	communicationHosts := make([]CommunicationHost, 0, len(resp.CommunicationEndpoints))
	formattedCommunicationEndpoints := resp.FormattedCommunicationEndpoints

	for _, s := range resp.CommunicationEndpoints {
		e, err := ParseEndpoint(s)
		if err != nil {
			log.Info("failed to parse communication endpoint", "url", s)
			continue
		}
		communicationHosts = append(communicationHosts, e)
	}

	ci := OneAgentConnectionInfo{
		CommunicationHosts: communicationHosts,
		ConnectionInfo: ConnectionInfo{
			TenantUUID:  tenantUUID,
			TenantToken: tenantToken,
			Endpoints:   formattedCommunicationEndpoints,
		},
	}
	return ci, nil
}

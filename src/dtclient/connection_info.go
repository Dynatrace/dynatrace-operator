package dtclient

import (
	"encoding/json"
	"github.com/pkg/errors"
)

type ConnectionInfo struct {
	TenantUUID  string
	TenantToken string
	Endpoints   string
}

type ActiveGateConnectionInfo struct {
	ConnectionInfo
}

type OneAgentConnectionInfo struct {
	ConnectionInfo
	CommunicationHosts []CommunicationHost
}

// CommunicationHost => struct of connection endpoint
type CommunicationHost struct {
	Protocol string
	Host     string
	Port     uint32
}

func (dtc *dynatraceClient) GetActiveGateConnectionInfo() (*ActiveGateConnectionInfo, error) {
	response, err := dtc.makeRequest(
		dtc.getActiveGateConnectionInfoUrl(),
		dynatracePaaSToken,
	)

	if err != nil {
		return nil, errors.WithStack(err)
	}

	defer func() {
		err := response.Body.Close()
		if err != nil {
			log.Error(err, err.Error())
		}
	}()

	data, err := dtc.getServerResponseData(response)
	if err != nil {
		return nil, dtc.handleErrorResponseFromAPI(data, response.StatusCode)
	}

	tenantInfo, err := dtc.readResponseForActiveGateTenantInfo(data)
	if err != nil {
		return nil, err
	}
	if len(tenantInfo.Endpoints) == 0 {
		log.Info("tenant has no endpoints")
	}

	return tenantInfo, nil
}

type activeGateConnectionInfoJsonResponse struct {
	TenantUUID             string `json:"tenantUUID"`
	TenantToken            string `json:"tenantToken"`
	CommunicationEndpoints string `json:"communicationEndpoints"`
}

func (dtc *dynatraceClient) readResponseForActiveGateTenantInfo(response []byte) (*ActiveGateConnectionInfo, error) {
	resp := activeGateConnectionInfoJsonResponse{}
	err := json.Unmarshal(response, &resp)
	if err != nil {
		log.Error(err, "error unmarshalling activegate tenant info", "response", string(response))
		return nil, err
	}

	agTenantInfo := &ActiveGateConnectionInfo{
		ConnectionInfo: ConnectionInfo{
			TenantUUID:  resp.TenantUUID,
			TenantToken: resp.TenantToken,
			Endpoints:   resp.CommunicationEndpoints,
		},
	}
	return agTenantInfo, nil
}

func (dtc *dynatraceClient) GetOneAgentConnectionInfo() (OneAgentConnectionInfo, error) {
	resp, err := dtc.makeRequest(dtc.getOneAgentConnectionInfoUrl(), dynatracePaaSToken)
	if err != nil {
		return OneAgentConnectionInfo{}, err
	}
	defer func() {
		//Swallow error, nothing has to be done at this point
		_ = resp.Body.Close()
	}()

	responseData, err := dtc.getServerResponseData(resp)
	if err != nil {
		return OneAgentConnectionInfo{}, err
	}

	return dtc.readResponseForOneAgentConnectionInfo(responseData)
}

func (dtc *dynatraceClient) readResponseForOneAgentConnectionInfo(response []byte) (OneAgentConnectionInfo, error) {
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
		return OneAgentConnectionInfo{}, err
	}

	tenantUuid := resp.TenantUUID
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

	if len(communicationHosts) == 0 {
		return OneAgentConnectionInfo{}, errors.New("no communication hosts available")
	}

	ci := OneAgentConnectionInfo{
		CommunicationHosts: communicationHosts,
		ConnectionInfo: ConnectionInfo{
			TenantUUID:  tenantUuid,
			TenantToken: tenantToken,
			Endpoints:   formattedCommunicationEndpoints,
		},
	}

	return ci, nil
}

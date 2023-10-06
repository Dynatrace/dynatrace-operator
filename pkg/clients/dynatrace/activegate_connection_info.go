package dynatrace

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

type activeGateConnectionInfoJsonResponse struct {
	TenantUUID             string `json:"tenantUUID"`
	TenantToken            string `json:"tenantToken"`
	CommunicationEndpoints string `json:"communicationEndpoints"`
}

func (dtc *dynatraceClient) GetActiveGateConnectionInfo() (ActiveGateConnectionInfo, error) {
	response, err := dtc.makeRequest(
		dtc.getActiveGateConnectionInfoUrl(),
		dynatracePaaSToken,
	)

	if err != nil {
		return ActiveGateConnectionInfo{}, errors.WithStack(err)
	}

	defer CloseBodyAfterRequest(response)

	data, err := dtc.getServerResponseData(response)
	if err != nil {
		return ActiveGateConnectionInfo{}, dtc.handleErrorResponseFromAPI(data, response.StatusCode)
	}

	tenantInfo, err := dtc.readResponseForActiveGateTenantInfo(data)
	if err != nil {
		return ActiveGateConnectionInfo{}, err
	}

	if len(tenantInfo.Endpoints) == 0 {
		log.Info("tenant has no endpoints")
	}

	return tenantInfo, nil
}

func (dtc *dynatraceClient) readResponseForActiveGateTenantInfo(response []byte) (ActiveGateConnectionInfo, error) {
	resp := activeGateConnectionInfoJsonResponse{}
	err := json.Unmarshal(response, &resp)
	if err != nil {
		log.Error(err, "error unmarshalling activegate tenant info", "response", string(response))
		return ActiveGateConnectionInfo{}, err
	}

	agTenantInfo := ActiveGateConnectionInfo{
		ConnectionInfo: ConnectionInfo{
			TenantUUID:  resp.TenantUUID,
			TenantToken: resp.TenantToken,
			Endpoints:   resp.CommunicationEndpoints,
		},
	}
	return agTenantInfo, nil
}

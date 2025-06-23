package dynatrace

import (
	"context"
	"encoding/json"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/utils"
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

type activeGateConnectionInfoJSONResponse struct {
	TenantUUID             string `json:"tenantUUID"`
	TenantToken            string `json:"tenantToken"`
	CommunicationEndpoints string `json:"communicationEndpoints"`
}

func (dtc *dynatraceClient) GetActiveGateConnectionInfo(ctx context.Context) (ActiveGateConnectionInfo, error) {
	response, err := dtc.makeRequest(
		ctx,
		dtc.getActiveGateConnectionInfoURL(),
		dynatracePaaSToken,
	)
	defer utils.CloseBodyAfterRequest(response)

	if err != nil {
		return ActiveGateConnectionInfo{}, errors.WithStack(err)
	}

	data, err := dtc.getServerResponseData(response)
	if err != nil {
		return ActiveGateConnectionInfo{}, dtc.handleErrorResponseFromAPI(data, response.StatusCode, response.Header)
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
	resp := activeGateConnectionInfoJSONResponse{}

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

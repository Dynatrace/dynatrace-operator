package dynatrace

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/utils"
)

type OneAgentConnectionInfo struct {
	ConnectionInfo
}

type oneAgentConnectionInfoJSONResponse struct {
	TenantUUID                      string   `json:"tenantUUID"`
	TenantToken                     string   `json:"tenantToken"`
	FormattedCommunicationEndpoints string   `json:"formattedCommunicationEndpoints"`
	CommunicationEndpoints          []string `json:"communicationEndpoints"`
}

func (dtc *dynatraceClient) GetOneAgentConnectionInfo(ctx context.Context) (OneAgentConnectionInfo, error) {
	resp, err := dtc.makeRequest(ctx, dtc.getOneAgentConnectionInfoURL(), dynatracePaaSToken)
	if err != nil {
		return OneAgentConnectionInfo{}, err
	}

	defer utils.CloseBodyAfterRequest(resp)

	if resp.StatusCode == http.StatusBadRequest {
		log.Info("server could not find the network zone or deliver default fallback config, is there an ActiveGate configured for the network zone?")

		return OneAgentConnectionInfo{}, nil
	}

	responseData, err := dtc.getServerResponseData(resp)
	if err != nil {
		return OneAgentConnectionInfo{}, dtc.handleErrorResponseFromAPI(responseData, resp.StatusCode, resp.Header)
	}

	connectionInfo, err := dtc.readResponseForOneAgentConnectionInfo(responseData)
	if err != nil {
		return OneAgentConnectionInfo{}, err
	}

	return connectionInfo, nil
}

func (dtc *dynatraceClient) readResponseForOneAgentConnectionInfo(response []byte) (OneAgentConnectionInfo, error) {
	resp := oneAgentConnectionInfoJSONResponse{}

	err := json.Unmarshal(response, &resp)
	if err != nil {
		log.Error(err, "error unmarshalling connection info response", "response", string(response))

		return OneAgentConnectionInfo{}, err
	}

	tenantUUID := resp.TenantUUID
	tenantToken := resp.TenantToken
	formattedCommunicationEndpoints := resp.FormattedCommunicationEndpoints

	ci := OneAgentConnectionInfo{
		ConnectionInfo: ConnectionInfo{
			TenantUUID:  tenantUUID,
			TenantToken: tenantToken,
			Endpoints:   formattedCommunicationEndpoints,
		},
	}

	return ci, nil
}

package oneagent

import (
	"context"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	"github.com/pkg/errors"
)

type ConnectionInfo struct {
	TenantUUID  string
	TenantToken string
	Endpoints   string
}

type connectionInfoJSONResponse struct {
	TenantUUID                      string   `json:"tenantUUID"`
	TenantToken                     string   `json:"tenantToken"`
	FormattedCommunicationEndpoints string   `json:"formattedCommunicationEndpoints"`
	CommunicationEndpoints          []string `json:"communicationEndpoints"`
}

func (c *Client) GetConnectionInfo(ctx context.Context) (ConnectionInfo, error) {
	var resp connectionInfoJSONResponse

	err := c.apiClient.GET(ctx, getOneAgentConnectionInfoURL(c.networkZone)).
		WithPaasToken().
		Execute(&resp)

	if core.IsBadRequest(err) {
		log.Info("server could not find the network zone or deliver default fallback config, is there an ActiveGate configured for the network zone?")

		return ConnectionInfo{}, nil
	}

	if err != nil {
		return ConnectionInfo{}, errors.WithStack(err)
	}

	connectionInfo := ConnectionInfo{
		TenantUUID:  resp.TenantUUID,
		TenantToken: resp.TenantToken,
		Endpoints:   resp.FormattedCommunicationEndpoints,
	}

	return connectionInfo, nil
}

func getOneAgentConnectionInfoURL(networkZone string) string {
	if networkZone != "" {
		return fmt.Sprintf("/v1/deployment/installer/agent/connectioninfo?networkZone=%s&defaultZoneFallback=true", networkZone)
	}

	return "/v1/deployment/installer/agent/connectioninfo"
}

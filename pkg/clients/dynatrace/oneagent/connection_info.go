package oneagent

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	"github.com/pkg/errors"
)

const (
	connectionInfoPath = "/v1/deployment/installer/agent/connectioninfo"
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

	params := map[string]string{}
	if c.networkZone != "" {
		params["networkZone"] = c.networkZone
		params["defaultZoneFallback"] = "true"
	}

	err := c.apiClient.GET(ctx, connectionInfoPath).
		WithQueryParams(params).
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

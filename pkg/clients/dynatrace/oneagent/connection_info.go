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
	TenantUUID  string `json:"tenantUUID"`
	TenantToken string `json:"tenantToken"`
	Endpoints   string `json:"formattedCommunicationEndpoints"`
}

func (c *Client) GetConnectionInfo(ctx context.Context) (ConnectionInfo, error) {
	var resp ConnectionInfo

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

	return resp, nil
}

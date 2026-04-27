package oneagent

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/pkg/errors"
)

const (
	connectionInfoPath = "/v1/deployment/installer/agent/connectioninfo"
)

type ConnectionInfo struct {
	TenantUUID  string `json:"tenantUUID"`
	TenantToken string `json:"tenantToken"`
	Endpoints   string `json:"formattedCommunicationEndpoints"`
	// NOTE: connectionInfoPath also returns
	// communicationEndpoints []string (individual endpoints as a slice), but we only
	// use the pre-formatted Endpoints string above. The slice is available if needed in the future.
}

func (c *ClientImpl) GetConnectionInfo(ctx context.Context) (ConnectionInfo, error) {
	ctx, log := logd.NewFromContext(ctx, loggerName)
	var resp ConnectionInfo

	params := map[string]string{}
	if c.networkZone != "" {
		params["networkZone"] = c.networkZone
		params["defaultZoneFallback"] = "true"
	}

	err := c.apiClient.GET(ctx, connectionInfoPath).
		WithPaasToken().
		WithQueryParams(params).
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

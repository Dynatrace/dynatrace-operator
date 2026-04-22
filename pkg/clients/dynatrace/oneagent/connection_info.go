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
	// NOTE: connectionInfoPath also returns
	// communicationEndpoints []string (individual endpoints as a slice), but we only
	// use the pre-formatted Endpoints string above. The slice is available if needed in the future.
}

func (c *Client) GetConnectionInfo(ctx context.Context) (ConnectionInfo, error) {
	var resp ConnectionInfo

	req := c.apiClient.GET(ctx, connectionInfoPath).WithPaasToken()

	params := map[string]string{}
	if c.networkZone != "" {
		params["networkZone"] = c.networkZone
		params["defaultZoneFallback"] = "true"

		// Skip the cache: when a network zone is set, the API may return empty Endpoints while
		// the ActiveGate that owns the zone is still being deployed. The client treats an empty Endpoints response
		// as valid, so only the reconciler detects the missing endpoints, by then the empty result has been cached
		// which would cause repeated no-ops until the TTL expires.
		req = req.WithSkipCache()
	}

	err := req.WithQueryParams(params).Execute(&resp)

	if core.IsBadRequest(err) {
		log.Info("server could not find the network zone or deliver default fallback config, is there an ActiveGate configured for the network zone?")

		return ConnectionInfo{}, nil
	}

	if err != nil {
		return ConnectionInfo{}, errors.WithStack(err)
	}

	return resp, nil
}

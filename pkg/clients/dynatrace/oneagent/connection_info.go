// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package oneagent

import (
	"context"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/pkg/errors"
)

const (
	connectionInfoPath = "/v1/deployment/installer/agent/connectioninfo"

	// endpointDelimiter separates the individual endpoints inside the
	// formattedCommunicationEndpoints string returned by the connectioninfo API.
	// It matches the delimiter used by connectioninfo.ParseOACommunicationHosts.
	endpointDelimiter = ";"
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

	resp.Endpoints = deduplicateEndpoints(resp.Endpoints)

	return resp, nil
}

// deduplicateEndpoints removes duplicate endpoints from a formatted
// communication-endpoints string (delimiter-separated, see endpointDelimiter).
func deduplicateEndpoints(endpoints string) string {
	if endpoints == "" {
		return endpoints
	}

	seen := make(map[string]struct{})
	unique := make([]string, 0)

	for endpoint := range strings.SplitSeq(endpoints, endpointDelimiter) {
		endpoint = strings.TrimSpace(endpoint)
		if endpoint == "" {
			continue
		}

		if _, ok := seen[endpoint]; ok {
			continue
		}

		seen[endpoint] = struct{}{}
		unique = append(unique, endpoint)
	}

	return strings.Join(unique, endpointDelimiter)
}

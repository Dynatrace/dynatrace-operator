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
)

// connectionInfoResponse is the raw shape returned by the connectioninfo API.
// It is unmarshalled directly from JSON and never exposed outside this package.
type connectionInfoResponse struct {
	TenantUUID             string   `json:"tenantUUID"`
	TenantToken            string   `json:"tenantToken"`
	CommunicationEndpoints []string `json:"communicationEndpoints"`
}

// ConnectionInfo is the public result of GetConnectionInfo. Endpoints is
// always deduplicated and semicolon-separated, consumers never see the raw
// endpoint list returned by the API
type ConnectionInfo struct {
	TenantUUID  string
	TenantToken string
	Endpoints   string
}

func (c *ClientImpl) GetConnectionInfo(ctx context.Context) (ConnectionInfo, error) {
	ctx, log := logd.NewFromContext(ctx, loggerName)

	var resp connectionInfoResponse

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

	return ConnectionInfo{
		TenantUUID:  resp.TenantUUID,
		TenantToken: resp.TenantToken,
		Endpoints:   deduplicateEndpoints(resp.CommunicationEndpoints),
	}, nil
}

// deduplicateEndpoints removes duplicate entries from the communication
// endpoints returned by the API and joins the unique ones into a single,
// comma-separated string. The order of first occurrence is preserved,
// surrounding whitespace is trimmed and empty entries are dropped.
func deduplicateEndpoints(endpoints []string) string {
	seen := make(map[string]struct{})

	var unique []string

	for _, endpoint := range endpoints {
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

	return strings.Join(unique, ";")
}

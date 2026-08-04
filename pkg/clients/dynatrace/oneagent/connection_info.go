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

type ConnectionInfo struct {
	TenantUUID             string   `json:"tenantUUID"`
	TenantToken            string   `json:"tenantToken"`
	Endpoints              string   `json:"formattedCommunicationEndpoints"`
	CommunicationEndpoints []string `json:"communicationEndpoints"`
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

	resp.Endpoints = deduplicateEndpoints(resp.CommunicationEndpoints)

	return resp, nil
}

// deduplicateEndpoints removes duplicate entries from the communication
// endpoints returned by the API and joins the unique ones into a single,
// comma-separated string. The order of first occurrence is preserved,
// surrounding whitespace is trimmed and empty entries are dropped.
func deduplicateEndpoints(endpoints []string) string {
	seen := make(map[string]struct{})
	unique := make([]string, 0)

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

	return strings.Join(unique, ",")
}

// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package oneagent

import (
	"context"
	"slices"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/pkg/errors"
)

const (
	connectionInfoPath = "/v1/deployment/installer/agent/connectioninfo"
)

var (
	NoCommunicationEndpointsError  = errors.New("no communication endpoints for OneAgent are available")
	StaleNetworkZoneEndpointsError = errors.New("OneAgent endpoints do not contain the local ActiveGate Service IP, waiting for the ActiveGate to register itself")
)

// connectionInfoResponse is the raw shape returned by the connectioninfo API.
// It is unmarshalled directly from JSON and never exposed outside this package.
// It implements core.Cacheable so that Execute opts the request into the HTTP-layer cache.
type connectionInfoResponse struct {
	TenantUUID             string   `json:"tenantUUID"`
	TenantToken            string   `json:"tenantToken"`
	CommunicationEndpoints []string `json:"communicationEndpoints"`
}

func (r *connectionInfoResponse) IsEmpty() bool {
	return r.TenantUUID == "" || r.TenantToken == "" || len(r.CommunicationEndpoints) == 0
}

// ConnectionInfo is the public result of GetConnectionInfo. Endpoints is
// always deduplicated and semicolon-separated, consumers never see the raw
// endpoint list returned by the API
type ConnectionInfo struct {
	TenantUUID  string
	TenantToken string
	Endpoints   string
}

func (cinf *ConnectionInfo) IsEmpty() bool {
	return cinf.TenantUUID == "" || cinf.TenantToken == "" || cinf.Endpoints == ""
}

func (c *ClientImpl) GetConnectionInfo(ctx context.Context, requiredHosts []string) (ConnectionInfo, error) {
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

	endpoints, err := buildEndpoints(resp.CommunicationEndpoints, requiredHosts)

	result := ConnectionInfo{
		TenantUUID:  resp.TenantUUID,
		TenantToken: resp.TenantToken,
		Endpoints:   endpoints,
	}

	return result, err
}

func buildEndpoints(rawEndpoints []string, requiredHosts []string) (string, error) {
	if len(rawEndpoints) == 0 {
		return "", NoCommunicationEndpointsError
	}

	seen := make(map[string]struct{})
	missing := len(requiredHosts) > 0

	var unique []string

	for _, endpoint := range rawEndpoints {
		endpoint = strings.TrimSpace(endpoint)
		if endpoint == "" {
			continue
		}

		if _, ok := seen[endpoint]; ok {
			continue
		}

		seen[endpoint] = struct{}{}
		unique = append(unique, endpoint)

		if missing {
			if ch, err := connectioninfo.NewCommunicationHost(endpoint); err == nil {
				missing = !slices.Contains(requiredHosts, ch.Host)
			}
		}
	}

	joined := strings.Join(unique, ";")

	if missing {
		return joined, StaleNetworkZoneEndpointsError
	}

	return joined, nil
}

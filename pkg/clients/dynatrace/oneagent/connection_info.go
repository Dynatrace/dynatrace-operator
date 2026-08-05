// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package oneagent

import (
	"context"
	"fmt"
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

// connectionInfoResponse is the raw shape returned by the connectioninfo API.
// It is unmarshalled directly from JSON and never exposed outside this package.
type connectionInfoResponse struct {
	TenantUUID             string   `json:"tenantUUID"`
	TenantToken            string   `json:"tenantToken"`
	CommunicationEndpoints []string `json:"communicationEndpoints"`
}

// MissingIPs is returned by GetConnectionInfo when provided required IPs are absent
// from the cluster's response.
//
// This arises in network-zone scenarios where the DynaKube has a routing ActiveGate:
// the cluster is expected to return hosts that include every ClusterIP of the local
// ActiveGate Service. If any ClusterIP is absent, the cluster is still advertising a
// stale IP — typically right after the AG Service IPs changed and before the AG has
// re-registered (which can happen if the DynaKube was removed then reapplied).
// Propagating stale hosts to the OneAgent would leave it unable to reach the AG, so
// callers should postpone the OneAgent deployment until the cluster reissues correct
// hosts.
//
// GetConnectionInfo still returns the partial ConnectionInfo alongside this error so
// callers can log or inspect what the cluster actually returned.
//
// Note: the ClusterIP Service we create for the ActiveGate should have a single
// ClusterIP because we do not set spec.ipFamilyPolicy and that defaults to SingleStack.
// If for some reason the Service ended up DualStack the ActiveGate would register two
// ClusterIPs, one per IP family, and this case is handled as well.
type MissingIPs struct {
	endpoints   string
	requiredIPs []string
}

func (err MissingIPs) Error() string {
	return fmt.Sprintf("some IPs (%s) are missing from the requested connection info's endpoints: %s", err.requiredIPs, err.endpoints)
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

func (c *ClientImpl) GetConnectionInfo(ctx context.Context, requiredIPs []string) (ConnectionInfo, error) {
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

	if missingIPs(resp.Endpoints, requiredIPs) {
		err := MissingIPs{endpoints: resp.Endpoints, requiredIPs: requiredIPs}
		log.Debug(err.Error())

		return resp, err
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

func missingIPs(endpoints string, requiredIPs []string) bool {
	if len(requiredIPs) == 0 {
		return false
	}

	hosts, err := connectioninfo.ParseOACommunicationHosts(endpoints)
	if err != nil {
		return false
	}

	for _, ip := range requiredIPs {
		if !slices.ContainsFunc(hosts, func(h connectioninfo.CommunicationHost) bool { return h.Host == ip }) {
			return true
		}
	}

	return false
}

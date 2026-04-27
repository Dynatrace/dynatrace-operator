package token

import (
	"context"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
)

const lookupPath = "/v2/apiTokens/lookup"

const (
	ConditionTypeAPITokenSettingsRead  = "ApiTokenSettingsRead"
	ConditionTypeAPITokenSettingsWrite = "ApiTokenSettingsWrite"

	ScopeActiveGateTokenCreate    = "activeGateTokenManagement.create"
	ScopeDataExport               = "DataExport"
	ScopeInstallerDownload        = "InstallerDownload"
	ScopeLogsIngest               = "logs.ingest"
	ScopeMetricsIngest            = "metrics.ingest"
	ScopeOpenTelemetryTraceIngest = "openTelemetryTrace.ingest"
	ScopeSettingsRead             = "settings.read"
	ScopeSettingsWrite            = "settings.write"
)

var (
	OptionalScopes = map[string]string{
		ScopeSettingsRead:  ConditionTypeAPITokenSettingsRead,
		ScopeSettingsWrite: ConditionTypeAPITokenSettingsWrite,
	}

	_ core.Cacheable = &scopesResponse{}
)

type APIClient interface {
	GetScopes(ctx context.Context, token string) ([]string, error)
}

type Client struct {
	apiClient core.APIClient
}

type lookupRequest struct {
	Token string `json:"token"`
}

type scopesResponse struct {
	Scopes []string `json:"scopes"`
}

// IsEmpty always returns false: an empty scope list is a valid, cacheable response.
// It means the token has no scopes assigned, which is a configuration error in the Operator.
// The cache is automatically invalidated when the user updates their token.
func (s *scopesResponse) IsEmpty() bool {
	return false
}

func NewClient(apiClient core.APIClient) *Client {
	return &Client{
		apiClient: apiClient,
	}
}

func (c *Client) GetScopes(ctx context.Context, token string) ([]string, error) {
	req := lookupRequest{Token: token}

	var resp scopesResponse

	err := c.apiClient.POST(ctx, lookupPath).WithJSONBody(req).Execute(&resp)
	if err != nil {
		return nil, fmt.Errorf("get token scopes: %w", err)
	}

	return resp.Scopes, nil
}

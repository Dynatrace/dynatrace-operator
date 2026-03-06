package activegate

import (
	"context"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	"github.com/pkg/errors"
)

const (
	authTokenValidity = time.Hour * 24 * 60
	activeGateType    = "ENVIRONMENT"

	authTokenPath      = "/v2/activeGateTokens"
	connectionInfoPath = "/v1/deployment/installer/gateway/connectioninfo"
)

type APIClient interface {
	GetAuthToken(ctx context.Context, dynakubeName string) (*AuthTokenInfo, error)
	GetConnectionInfo(ctx context.Context) (ConnectionInfo, error)
}

type Client struct {
	apiClient core.APIClient
}

func NewClient(apiClient core.APIClient) *Client {
	return &Client{
		apiClient: apiClient,
	}
}

type AuthTokenInfo struct {
	TokenID string `json:"id"`
	Token   string `json:"token"`
}

type authTokenParams struct {
	Name           string `json:"name"`
	ActiveGateType string `json:"activeGateType"`
	ExpirationDate string `json:"expirationDate"`
	SeedToken      bool   `json:"seedToken"`
}

func (c *Client) GetAuthToken(ctx context.Context, dynakubeName string) (*AuthTokenInfo, error) {
	body := authTokenParams{
		Name:           dynakubeName,
		SeedToken:      false,
		ActiveGateType: activeGateType,
		ExpirationDate: getAuthTokenExpirationDate(),
	}

	var authTokenInfo AuthTokenInfo

	err := c.apiClient.POST(ctx, authTokenPath).
		WithJSONBody(body).
		Execute(&authTokenInfo)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to retrieve ag-auth-token")
	}

	return &authTokenInfo, nil
}

func getAuthTokenExpirationDate() string {
	return time.Now().Add(authTokenValidity).UTC().Format(time.RFC3339)
}

type ConnectionInfo struct {
	TenantUUID  string
	TenantToken string
	Endpoints   string
}

type connectionInfoJSONResponse struct {
	TenantUUID             string `json:"tenantUUID"`
	TenantToken            string `json:"tenantToken"`
	CommunicationEndpoints string `json:"communicationEndpoints"`
}

func (c *Client) GetConnectionInfo(ctx context.Context) (ConnectionInfo, error) {
	var resp connectionInfoJSONResponse

	err := c.apiClient.GET(ctx, connectionInfoPath).
		WithPaasToken().
		Execute(&resp)
	if err != nil {
		return ConnectionInfo{}, errors.WithStack(err)
	}

	connectionInfo := ConnectionInfo{
		TenantUUID:  resp.TenantUUID,
		TenantToken: resp.TenantToken,
		Endpoints:   resp.CommunicationEndpoints,
	}

	return connectionInfo, nil
}

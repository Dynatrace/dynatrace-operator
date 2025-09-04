package token

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace4/core"
)

type client struct {
	apiClient core.APIClient
}

var _ Client = (*client)(nil)

// NewClient creates a new Token API client
func NewClient(apiClient core.APIClient) Client {
	return &client{
		apiClient: apiClient,
	}
}

// TokenScopes is a list of scopes assigned to a token
type TokenScopes []string

// GetTokenScopes returns the list of scopes assigned to a token if successful.
func (c *client) GetTokenScopes(ctx context.Context, token string) (TokenScopes, error) {
	if token == "" {
		return nil, nil
	}

	body := map[string]string{"token": token}

	var resp struct {
		Scopes []string `json:"scopes"`
	}

	err := c.apiClient.POST(APITokenPath).
		WithContext(ctx).
		WithJSONBody(body).
		Execute(&resp)
	if err != nil {
		return nil, err
	}

	return resp.Scopes, nil
}

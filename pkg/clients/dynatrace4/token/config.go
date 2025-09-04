package token

import (
	"context"
)

const (
	APITokenPath = "/v2/apiTokens/lookup"
)

type Client interface {
	GetTokenScopes(ctx context.Context, token string) (TokenScopes, error)
}

package token

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

var (
	log = logd.Get().WithName("dtclient.token")
)

const (
	ApiTokenPath = "/v2/apiTokens/lookup"
)

type Client interface {
	GetTokenScopes(ctx context.Context, token string) (TokenScopes, error)
}

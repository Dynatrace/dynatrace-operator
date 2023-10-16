package edgeconnect

import (
	"context"
	"net/http"

	"golang.org/x/oauth2/clientcredentials"
)

func NewCredentialsClient(ctx context.Context, clientID, clientSecret string, scopes []string, tokenURL string) *http.Client {
	conf := &clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       scopes,
		TokenURL:     tokenURL,
	}
	return conf.Client(ctx)
}

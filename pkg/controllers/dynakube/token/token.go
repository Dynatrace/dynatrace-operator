package token

import (
	"context"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/sets"
)

type Token struct {
	Type     string
	Value    string
	Features []Feature
}

func newToken(tokenType string, value string) Token {
	return Token{
		Type:     tokenType,
		Value:    value,
		Features: make([]Feature, 0),
	}
}

func (token *Token) addFeatures(features []Feature) {
	token.Features = append(token.Features, features...)
}

func (token *Token) verifyScopes(ctx context.Context, dtClient dtclient.Client, dk dynakube.DynaKube) ([]string, error) {
	if len(token.Features) == 0 {
		return nil, nil
	}

	scopes, err := dtClient.GetTokenScopes(ctx, token.Value)
	if err != nil {
		return nil, err
	}

	err = token.verifyRequiredScopes(scopes, dk)

	missingOptionalScopes := token.collectMissingOptionalScopes(scopes, dk)

	if len(missingOptionalScopes) > 0 {
		log.Info("some optional scopes are missing", "missing scopes", missingOptionalScopes, "token", token.Type)
	}

	return missingOptionalScopes, err
}

func (token *Token) verifyRequiredScopes(scopes dtclient.TokenScopes, dk dynakube.DynaKube) error {
	collectedErrors := make([]error, 0)

	for _, feature := range token.Features {
		if feature.IsEnabled(dk) {
			missingScopes := feature.CollectMissingRequiredScopes(scopes)
			if len(missingScopes) > 0 {
				collectedErrors = append(collectedErrors,
					errors.Errorf("feature '%s' is missing scope '%s'",
						feature.Name,
						strings.Join(missingScopes, ", ")))
			}
		}
	}

	if len(collectedErrors) > 0 {
		return errors.Errorf("token '%s' has scope errors: %s", token.Type, collectedErrors)
	}

	return nil
}

func (token *Token) collectMissingOptionalScopes(scopes dtclient.TokenScopes, dk dynakube.DynaKube) []string {
	missingScopes := sets.NewString()

	for _, feature := range token.Features {
		if feature.IsEnabled(dk) {
			missingScopes.Insert(feature.CollectMissingOptionalScopes(scopes)...)
		}
	}

	return missingScopes.List()
}

func (token *Token) verifyValue() error {
	if strings.TrimSpace(token.Value) != token.Value {
		return errors.Errorf("token '%s' contains leading or trailing whitespaces", token.Type)
	}

	return nil
}

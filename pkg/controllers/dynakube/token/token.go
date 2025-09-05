package token

import (
	"context"
	"maps"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/pkg/errors"
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

func (token *Token) verifyScopes(ctx context.Context, dtClient dtclient.Client, dk dynakube.DynaKube) (map[string]bool, error) {
	if len(token.Features) == 0 {
		return map[string]bool{}, nil
	}

	scopes, err := dtClient.GetTokenScopes(ctx, token.Value)
	if err != nil {
		return nil, err
	}

	err = token.verifyRequiredScopes(scopes, dk)

	optionalScopes := token.collectOptionalScopes(scopes, dk)

	missingOptionalScopes := []string{}

	for scope, isAvailable := range optionalScopes {
		if !isAvailable {
			missingOptionalScopes = append(missingOptionalScopes, scope)
		}
	}

	if len(missingOptionalScopes) > 0 {
		log.Info("some optional scopes are missing", "missing scopes", missingOptionalScopes, "token", token.Type)
	}

	return optionalScopes, err
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

func (token *Token) collectOptionalScopes(availableScopes dtclient.TokenScopes, dk dynakube.DynaKube) map[string]bool {
	optionalScopes := map[string]bool{}

	for _, feature := range token.Features {
		if feature.IsEnabled(dk) {
			maps.Insert(optionalScopes, maps.All(feature.CollectOptionalScopes(availableScopes)))
		}
	}

	return optionalScopes
}

func (token *Token) verifyValue() error {
	if strings.TrimSpace(token.Value) != token.Value {
		return errors.Errorf("token '%s' contains leading or trailing whitespaces", token.Type)
	}

	return nil
}

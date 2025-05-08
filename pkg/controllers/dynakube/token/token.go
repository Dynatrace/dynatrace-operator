package token

import (
	"context"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta5/dynakube"
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

func (token *Token) verifyScopes(ctx context.Context, dtClient dtclient.Client, dk dynakube.DynaKube) error {
	if len(token.Features) == 0 {
		return nil
	}

	scopes, err := dtClient.GetTokenScopes(ctx, token.Value)
	if err != nil {
		return err
	}

	collectedErrors := make([]error, 0)

	for _, feature := range token.Features {
		if feature.IsEnabled(dk) {
			isMissing, missingScopes := feature.IsScopeMissing(scopes)
			if isMissing {
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

func (token *Token) verifyValue() error {
	if strings.TrimSpace(token.Value) != token.Value {
		return errors.Errorf("token '%s' contains leading or trailing whitespaces", token.Type)
	}

	return nil
}

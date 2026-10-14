// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package token

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	tokenclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/pkg/errors"
)

type Token struct {
	Type     string
	Value    string
	Features []Feature
}

// OptionalScope describes an optional token scope.
// Available reflects whether the token has the scope, Requested whether an enabled feature uses it.
type OptionalScope struct {
	Available bool
	Requested bool
}

type ScopeError struct {
	Token         string
	MissingScopes []string
	Errs          []error
}

func (m ScopeError) Error() string {
	return fmt.Sprintf("token '%s' has scope errors: %v", m.Token, m.Errs)
}

func newToken(tokenType string, value string) Token {
	return Token{
		Type:     tokenType,
		Value:    value,
		Features: make([]Feature, 0),
	}
}

func (token *Token) String() string {
	if token == nil {
		return ""
	}

	return token.Value
}

func (token *Token) addFeatures(features []Feature) {
	token.Features = append(token.Features, features...)
}

func (token *Token) verifyScopes(ctx context.Context, dtClient tokenclient.Client, dk *dynakube.DynaKube) (map[string]OptionalScope, error) {
	ctx, log := logd.NewFromContext(ctx, "tokens")

	if len(token.Features) == 0 {
		return map[string]OptionalScope{}, nil
	}

	scopes, err := dtClient.GetScopes(ctx, token.Value)
	if err != nil {
		return nil, err
	}

	err = token.verifyRequiredScopes(scopes, dk)

	optionalScopes := token.collectOptionalScopes(scopes, dk)

	missingOptionalScopes := []string{}

	for scope, optionalScope := range optionalScopes {
		if optionalScope.Requested && !optionalScope.Available {
			missingOptionalScopes = append(missingOptionalScopes, scope)
		}
	}

	if len(missingOptionalScopes) > 0 {
		log.Info("some optional scopes are missing", "missing scopes", missingOptionalScopes, "token", token.Type)
	}

	return optionalScopes, err
}

func (token *Token) verifyRequiredScopes(scopes []string, dk *dynakube.DynaKube) error {
	collectedErrors := make([]error, 0)
	allMissingScopes := make([]string, 0)

	for _, feature := range token.Features {
		if feature.IsEnabled(dk) {
			missingScopes := feature.CollectMissingRequiredScopes(scopes)
			if len(missingScopes) > 0 {
				allMissingScopes = append(allMissingScopes, missingScopes...)
				collectedErrors = append(collectedErrors,
					errors.Errorf("feature '%s' is missing scope '%s'",
						feature.Name,
						strings.Join(missingScopes, ", ")))
			}
		}
	}

	if len(collectedErrors) > 0 {
		return ScopeError{Errs: collectedErrors, Token: token.Type, MissingScopes: allMissingScopes}
	}

	return nil
}

// collectOptionalScopes reports the availability of all optional scopes for the API token, independent of
// the enabled features, because some consumers (e.g. the MEID lookup) are not tied to a feature.
func (token *Token) collectOptionalScopes(availableScopes []string, dk *dynakube.DynaKube) map[string]OptionalScope {
	optionalScopes := map[string]OptionalScope{}
	if token.Type != APIKey {
		return optionalScopes
	}

	for _, scope := range tokenclient.OptionalScopes {
		optionalScopes[scope] = OptionalScope{Available: slices.Contains(availableScopes, scope)}
	}

	for _, feature := range token.Features {
		if !feature.IsEnabled(dk) {
			continue
		}

		for _, scope := range feature.OptionalScopes {
			optionalScope := optionalScopes[scope]
			optionalScope.Requested = true
			optionalScopes[scope] = optionalScope
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

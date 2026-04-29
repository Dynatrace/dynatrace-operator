package token

import (
	"context"
	"errors"
	"maps"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/token"
)

const (
	PaaSKey       = "paasToken"
	APIKey        = "apiToken"
	DataIngestKey = "dataIngestToken"
)

type Tokens map[string]*Token

func (tokens Tokens) APIToken() *Token {
	return tokens.getToken(APIKey)
}

func (tokens Tokens) PaasToken() *Token {
	return tokens.getToken(PaaSKey)
}

func (tokens Tokens) DataIngestToken() *Token {
	return tokens.getToken(DataIngestKey)
}

func (tokens Tokens) getToken(tokenName string) *Token {
	token, hasToken := tokens[tokenName]
	if !hasToken {
		token = &Token{}
	}

	return token
}

func (tokens Tokens) AddFeatureScopesToTokens() Tokens {
	_, hasPaasToken := tokens[PaaSKey]

	for _, token := range tokens {
		switch token.Type {
		case APIKey:
			token.addFeatures(getFeaturesForAPIToken(hasPaasToken))
		case PaaSKey:
			token.addFeatures(getFeaturesForPaaSToken())
		case DataIngestKey:
			token.addFeatures(getFeaturesForDataIngest())
		}
	}

	return tokens
}

func (tokens Tokens) VerifyScopes(ctx context.Context, dtClient token.Client, dk dynakube.DynaKube) (map[string]bool, error) {
	var err error

	collectedMissingOptionalScopes := map[string]bool{}

	for _, token := range tokens {
		missingOptionalScopes, scopeError := token.verifyScopes(ctx, dtClient, dk)
		if scopeError != nil {
			err = errors.Join(err, scopeError)
		}

		maps.Insert(collectedMissingOptionalScopes, maps.All(missingOptionalScopes))
	}

	return collectedMissingOptionalScopes, err
}

func (tokens Tokens) VerifyValues() (err error) {
	for _, token := range tokens {
		verifyErr := token.verifyValue()
		if verifyErr != nil {
			err = errors.Join(err, verifyErr)
		}
	}

	return
}

func CheckForDataIngestToken(tokens Tokens) bool {
	dataIngestToken, hasDataIngestToken := tokens[DataIngestKey]

	return hasDataIngestToken && len(dataIngestToken.Value) != 0
}

// GetMissingScopes inspects the provided error for ScopeError values and extracts their missing scopes.
// Returns a nil slice if no ScopeError is encountered.
func GetMissingScopes(err error) []string {
	if err == nil {
		return nil
	}

	var errs []error

	if unwrap, ok := err.(interface{ Unwrap() []error }); !ok {
		errs = []error{err}
	} else {
		errs = unwrap.Unwrap()
	}

	var missingScopes []string

	for _, err := range errs {
		if scopeErr := new(ScopeError); errors.As(err, scopeErr) {
			missingScopes = append(missingScopes, scopeErr.MissingScopes...)
		}
	}

	return missingScopes
}

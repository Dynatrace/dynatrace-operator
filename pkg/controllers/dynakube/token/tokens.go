package token

import (
	"context"
	"errors"
	"maps"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/dynatraceapi"
)

const (
	PaaSKey       = "paasToken"
	APIKey        = "apiToken"
	DataIngestKey = "dataIngestToken"
)

type VerificationError struct {
	Errs []error
}

func (v VerificationError) Error() string {
	return concatErrors(v.Errs).Error()
}

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

func (tokens Tokens) VerifyScopes(ctx context.Context, dtClient token.APIClient, dk dynakube.DynaKube) (map[string]bool, error) {
	collectedScopeErrors := make([]error, 0)
	collectedMissingOptionalScopes := map[string]bool{}

	for _, token := range tokens {
		missingOptionalScopes, scopeError := token.verifyScopes(ctx, dtClient, dk)
		if scopeError != nil {
			collectedScopeErrors = append(collectedScopeErrors, scopeError)
		}

		maps.Insert(collectedMissingOptionalScopes, maps.All(missingOptionalScopes))
	}

	if len(collectedScopeErrors) == 0 {
		return collectedMissingOptionalScopes, nil
	}

	return collectedMissingOptionalScopes, VerificationError{Errs: collectedScopeErrors}
}

func (tokens Tokens) VerifyValues() error {
	valueErrors := make([]error, 0)

	for _, token := range tokens {
		err := token.verifyValue()
		if err != nil {
			valueErrors = append(valueErrors, err)
		}
	}

	return concatErrors(valueErrors)
}

func concatErrors(errs []error) error {
	if len(errs) == 0 {
		return nil
	}

	apiStatus := dynatraceapi.NoError

	var concatenatedError strings.Builder
	for index, err := range errs {
		concatenatedError.WriteString(err.Error())

		if index < len(errs)-1 {
			concatenatedError.WriteString("\n\t")
		}

		if apiStatus == dynatraceapi.NoError && dynatraceapi.IsUnreachable(err) {
			apiStatus = dynatraceapi.StatusCode(err)
		}
	}

	if apiStatus != dynatraceapi.NoError {
		return core.ServerError{
			Code:    apiStatus,
			Message: concatenatedError.String(),
		}
	}

	return errors.New(concatenatedError.String())
}

func CheckForDataIngestToken(tokens Tokens) bool {
	dataIngestToken, hasDataIngestToken := tokens[DataIngestKey]

	return hasDataIngestToken && len(dataIngestToken.Value) != 0
}

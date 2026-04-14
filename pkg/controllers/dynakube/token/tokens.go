package token

import (
	"context"
	"errors"
	"maps"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/dynatraceapi"
)

const (
	PaasToken       = "paasToken"
	APIToken        = "apiToken"
	DataIngestToken = "dataIngestToken"
)

type VerificationError struct {
	Errs []error
}

func (v VerificationError) Error() string {
	return concatErrors(v.Errs).Error()
}

type Tokens map[string]*Token

func (tokens Tokens) APIToken() *Token {
	return tokens.getToken(APIToken)
}

func (tokens Tokens) PaasToken() *Token {
	return tokens.getToken(PaasToken)
}

func (tokens Tokens) DataIngestToken() *Token {
	return tokens.getToken(DataIngestToken)
}

func (tokens Tokens) getToken(tokenName string) *Token {
	token, hasToken := tokens[tokenName]
	if !hasToken {
		token = &Token{}
	}

	return token
}

func (tokens Tokens) AddFeatureScopesToTokens() Tokens {
	_, hasPaasToken := tokens[PaasToken]

	for _, token := range tokens {
		switch token.Type {
		case APIToken:
			token.addFeatures(getFeaturesForAPIToken(hasPaasToken))
		case PaasToken:
			token.addFeatures(getFeaturesForPaaSToken())
		case DataIngestToken:
			token.addFeatures(getFeaturesForDataIngest())
		}
	}

	return tokens
}

func (tokens Tokens) VerifyScopes(ctx context.Context, dtClient dtclient.Client, dk dynakube.DynaKube) (map[string]bool, error) {
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
		return dtclient.ServerError{
			Code:    apiStatus,
			Message: concatenatedError.String(),
		}
	}

	return errors.New(concatenatedError.String())
}

func CheckForDataIngestToken(tokens Tokens) bool {
	dataIngestToken, hasDataIngestToken := tokens[DataIngestToken]

	return hasDataIngestToken && len(dataIngestToken.Value) != 0
}

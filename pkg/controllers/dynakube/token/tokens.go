package token

import (
	"context"
	"errors"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/dynatraceapi"
)

type Tokens map[string]*Token

func (tokens Tokens) ApiToken() *Token {
	return tokens.getToken(dtclient.ApiToken)
}

func (tokens Tokens) PaasToken() *Token {
	return tokens.getToken(dtclient.PaasToken)
}

func (tokens Tokens) DataIngestToken() *Token {
	return tokens.getToken(dtclient.DataIngestToken)
}

func (tokens Tokens) getToken(tokenName string) *Token {
	token, hasToken := tokens[tokenName]
	if !hasToken {
		token = &Token{}
	}

	return token
}

func (tokens Tokens) AddFeatureScopesToTokens() Tokens {
	_, hasPaasToken := tokens[dtclient.PaasToken]

	for _, token := range tokens {
		switch token.Type {
		case dtclient.ApiToken:
			token.addFeatures(getFeaturesForAPIToken(hasPaasToken))
		case dtclient.PaasToken:
			token.addFeatures(getFeaturesForPaaSToken())
		case dtclient.DataIngestToken:
			token.addFeatures(getFeaturesForDataIngest())
		}
	}

	return tokens
}

func (tokens Tokens) VerifyScopes(ctx context.Context, dtClient dtclient.Client, dk dynakube.DynaKube) error {
	scopeErrors := make([]error, 0)

	for _, token := range tokens {
		if err := token.verifyScopes(ctx, dtClient, dk); err != nil {
			scopeErrors = append(scopeErrors, err)
		}
	}

	if len(scopeErrors) > 0 {
		return concatErrors(scopeErrors)
	}

	return nil
}

func (tokens Tokens) VerifyValues() error {
	valueErrors := make([]error, 0)

	for _, token := range tokens {
		err := token.verifyValue()
		if err != nil {
			valueErrors = append(valueErrors, err)
		}
	}

	if len(valueErrors) > 0 {
		return concatErrors(valueErrors)
	}

	return nil
}

func concatErrors(errs []error) error {
	concatenatedError := ""
	apiStatus := dynatraceapi.NoError

	for index, err := range errs {
		concatenatedError += err.Error()

		if index < len(errs)-1 {
			concatenatedError += "\n\t"
		}

		if apiStatus == dynatraceapi.NoError && dynatraceapi.IsUnreachable(err) {
			apiStatus = dynatraceapi.StatusCode(err)
		}
	}

	if apiStatus != dynatraceapi.NoError {
		return dtclient.ServerError{
			Code:    apiStatus,
			Message: concatenatedError,
		}
	}

	return errors.New(concatenatedError)
}

func CheckForDataIngestToken(tokens Tokens) bool {
	dataIngestToken, hasDataIngestToken := tokens[dtclient.DataIngestToken]

	return hasDataIngestToken && len(dataIngestToken.Value) != 0
}

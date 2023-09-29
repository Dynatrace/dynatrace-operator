package token

import (
	"fmt"
	"strings"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/dynatraceapi"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/pkg/errors"
)

type Tokens map[string]Token

func (tokens Tokens) ApiToken() Token {
	return tokens.getToken(dtclient.DynatraceApiToken)
}

func (tokens Tokens) PaasToken() Token {
	return tokens.getToken(dtclient.DynatracePaasToken)
}

func (tokens Tokens) DataIngestToken() Token {
	return tokens.getToken(dtclient.DynatraceDataIngestToken)
}

func (tokens Tokens) getToken(tokenName string) Token {
	token, hasToken := tokens[tokenName]
	if !hasToken {
		token = Token{}
	}

	return token
}

func (tokens Tokens) SetScopesForDynakube(dynakube dynatracev1beta1.DynaKube) Tokens {
	_, hasPaasToken := tokens[dtclient.DynatracePaasToken]

	for tokenType, token := range tokens {
		switch tokenType {
		case dtclient.DynatraceApiToken:
			tokens[dtclient.DynatraceApiToken] = token.setApiTokenScopes(dynakube, hasPaasToken)
		case dtclient.DynatracePaasToken:
			tokens[dtclient.DynatracePaasToken] = token.setPaasTokenScopes()
		case dtclient.DynatraceDataIngestToken:
			tokens[dtclient.DynatraceDataIngestToken] = token.setDataIngestScopes()
		}
	}

	return tokens
}

func (tokens Tokens) VerifyScopes(dtc dtclient.Client) error {
	scopeErrors := make([]error, 0)

	for tokenType, token := range tokens {
		if len(token.RequiredScopes) == 0 {
			continue
		}

		scopes, err := dtc.GetTokenScopes(token.Value)

		if err != nil {
			scopeErrors = append(scopeErrors, err)
			continue
		}

		missingScopes := token.getMissingScopes(scopes)

		if len(missingScopes) > 0 {
			scopeErrors = append(scopeErrors,
				errors.New(fmt.Sprintf("token '%s' is missing the following scopes: [ %s ]", tokenType, strings.Join(missingScopes, ", "))))
		}
	}

	if len(scopeErrors) > 0 {
		return concatErrors(scopeErrors)
	}

	return nil
}

func (tokens Tokens) VerifyValues() error {
	valueErrors := make([]error, 0)

	for tokenType, token := range tokens {
		if strings.TrimSpace(token.Value) != token.Value {
			valueErrors = append(valueErrors,
				errors.Errorf("value of token '%s' contains whitespaces at the beginning or end of the value", tokenType))
		}
	}

	if len(valueErrors) > 0 {
		return concatErrors(valueErrors)
	}

	return nil
}

func concatErrors(errs []error) error {
	concatenatedError := ""
	apiError := 0

	for index, err := range errs {
		concatenatedError += err.Error()

		if index < len(errs)-1 {
			concatenatedError += "\n\t"
		}

		if apiError == 0 && dynatraceapi.IsUnreachable(err) {
			apiError = dynatraceapi.StatusCode(err)
		}
	}

	if apiError != 0 {
		return dtclient.ServerError{
			Code:    apiError,
			Message: concatenatedError,
		}
	}
	return errors.New(concatenatedError)
}

package token

import (
	"fmt"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/pkg/errors"
	"strings"
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
		return Token{}
	}

	return token
}

func (tokens Tokens) setScopes(dynakube dynatracev1beta1.DynaKube) Tokens {
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

func (tokens Tokens) verifyScopes(dtc dtclient.Client) error {
	for tokenType, token := range tokens {
		if len(token.RequiredScopes) == 0 {
			continue
		}

		scopes, err := dtc.GetTokenScopes(token.Value)

		if err != nil {
			return err
		}

		missingScopes := token.getMissingScopes(scopes)

		if len(missingScopes) > 0 {
			return errors.New(fmt.Sprintf("token '%s' is missing the following scopes: [ %s ]", tokenType, strings.Join(missingScopes, ", ")))
		}
	}

	return nil
}

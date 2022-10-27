package token

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
)

// Constants are re-imported to avoid import cycle
// Todo: move constants to config package
const (
	DynatracePaasToken       = "paasToken"
	DynatraceApiToken        = "apiToken"
	DynatraceDataIngestToken = "dataIngestToken"
)

type Token struct {
	Value          string
	RequiredScopes []string
}

type Tokens map[string]Token

func (tokens Tokens) ApiToken() Token {
	return tokens.getToken(DynatraceApiToken)
}

func (tokens Tokens) PaasToken() Token {
	return tokens.getToken(DynatracePaasToken)
}

func (tokens Tokens) DataIngestToken() Token {
	return tokens.getToken(DynatraceDataIngestToken)
}

func (tokens Tokens) getToken(tokenName string) Token {
	token, hasToken := tokens[tokenName]

	if !hasToken {
		return Token{}
	}

	return token
}

func (tokens Tokens) setScopes(dynakube dynatracev1beta1.DynaKube) Tokens {

}

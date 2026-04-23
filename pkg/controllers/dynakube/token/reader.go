package token

import (
	"context"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/dttoken"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reader struct {
	apiReader client.Reader
	dk        *dynakube.DynaKube
}

func NewReader(apiReader client.Reader, dk *dynakube.DynaKube) Reader {
	return Reader{
		apiReader: apiReader,
		dk:        dk,
	}
}

// HasPlatformToken inspects the token secret and checks if the apiToken is a platform token.
// Returns an error if reading the secret fails.
func (reader Reader) HasPlatformToken(ctx context.Context) (bool, error) {
	tokens, err := reader.ReadTokens(ctx)
	if err != nil {
		return false, err
	}

	token := tokens.APIToken()
	if token != nil {
		return dttoken.IsPlatform(token.Value), nil
	}

	return false, nil
}

func (reader Reader) ReadAndVerifyTokens(ctx context.Context) (Tokens, error) {
	tokens, err := reader.ReadTokens(ctx)
	if err != nil {
		return nil, err
	}

	err = reader.verifyAPITokenExists(tokens)
	if err != nil {
		return nil, err
	}

	return tokens, nil
}

func (reader Reader) ReadTokens(ctx context.Context) (Tokens, error) {
	var tokenSecret corev1.Secret

	result := make(Tokens)

	err := reader.apiReader.Get(ctx, client.ObjectKey{
		Name:      reader.dk.Tokens(),
		Namespace: reader.dk.Namespace,
	}, &tokenSecret)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	for tokenType, rawToken := range tokenSecret.Data {
		token := newToken(tokenType, string(rawToken))
		result[tokenType] = &token
	}

	return result, nil
}

func (reader Reader) verifyAPITokenExists(tokens Tokens) error {
	apiToken, hasAPIToken := tokens[APIKey]

	if !hasAPIToken || len(apiToken.Value) == 0 {
		return errors.New(fmt.Sprintf("the API token is missing from the token secret '%s:%s'", reader.dk.Namespace, reader.dk.Tokens()))
	}

	return nil
}

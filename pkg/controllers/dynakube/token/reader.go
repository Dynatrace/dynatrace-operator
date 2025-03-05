package token

import (
	"context"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
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

func (reader Reader) ReadTokens(ctx context.Context) (Tokens, error) {
	tokens, err := reader.readTokens(ctx)
	if err != nil {
		return nil, err
	}

	err = reader.verifyApiTokenExists(tokens)
	if err != nil {
		return nil, err
	}

	return tokens, nil
}

func (reader Reader) readTokens(ctx context.Context) (Tokens, error) {
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

func (reader Reader) verifyApiTokenExists(tokens Tokens) error {
	apiToken, hasApiToken := tokens[dtclient.ApiToken]

	if !hasApiToken || len(apiToken.Value) == 0 {
		return errors.New(fmt.Sprintf("the API token is missing from the token secret '%s:%s'", reader.dk.Namespace, reader.dk.Tokens()))
	}

	return nil
}

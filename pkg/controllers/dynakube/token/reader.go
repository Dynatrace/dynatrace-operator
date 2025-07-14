package token

import (
	"context"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/dterror"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
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

	err = reader.verifyAPITokenExists(tokens)
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
		return nil, dterror.WithErrorCode(
			errors.WithMessage(errors.WithStack(err), "reading API token secret failed"),
			"DEC:xxxx")
	}

	for tokenType, rawToken := range tokenSecret.Data {
		token := newToken(tokenType, string(rawToken))
		result[tokenType] = &token
	}

	return result, nil
}

func (reader Reader) verifyAPITokenExists(tokens Tokens) error {
	apiToken, hasAPIToken := tokens[dtclient.APIToken]

	if !hasAPIToken || len(apiToken.Value) == 0 {
		return dterror.Errorf("DEC:C2", "the API token is missing in the token secret '%s:%s'", reader.dk.Namespace, reader.dk.Tokens())
	}

	return nil
}

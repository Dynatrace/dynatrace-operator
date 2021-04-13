package utils

import (
	"fmt"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

type Tokens struct {
	ApiToken  string
	PaasToken string
}

func NewTokens(secret *corev1.Secret) (*Tokens, error) {
	if secret == nil {
		return nil, fmt.Errorf("could not parse tokens: secret is nil")
	}

	var apiToken string
	var paasToken string
	var err error

	if err = verifySecret(secret); err != nil {
		return nil, errors.WithStack(err)
	}

	//Errors would have been caught by verifySecret
	apiToken, _ = ExtractToken(secret, dtclient.DynatraceApiToken)
	paasToken, _ = ExtractToken(secret, dtclient.DynatracePaasToken)

	return &Tokens{
		ApiToken:  apiToken,
		PaasToken: paasToken,
	}, nil
}

func verifySecret(secret *corev1.Secret) error {
	for _, token := range []string{
		dtclient.DynatraceApiToken,
		dtclient.DynatracePaasToken} {
		_, err := ExtractToken(secret, token)
		if err != nil {
			return errors.Errorf("invalid secret %s, %s", secret.Name, err)
		}
	}

	return nil
}

func ExtractToken(secret *corev1.Secret, key string) (string, error) {
	value, hasKey := secret.Data[key]
	if !hasKey {
		err := fmt.Errorf("missing token %s", key)
		return "", err
	}

	return strings.TrimSpace(string(value)), nil
}

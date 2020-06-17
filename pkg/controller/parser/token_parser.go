package parser

import (
	"fmt"
	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-activegate-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/controller/const"
	corev1 "k8s.io/api/core/v1"
	"strings"
)

type Tokens struct {
	ApiToken  string
	PaasToken string
}

func NewTokens(secret *corev1.Secret) (*Tokens, error) {
	var apiToken string
	var paasToken string
	var err error

	if err = verifySecret(secret); err != nil {
		return nil, err
	}

	apiToken, err = ExtractToken(secret, _const.DynatraceApiToken)
	if err != nil {
		return nil, err
	}

	paasToken, err = ExtractToken(secret, _const.DynatracePaasToken)
	if err != nil {
		return nil, err
	}

	return &Tokens{
		ApiToken:  apiToken,
		PaasToken: paasToken,
	}, nil
}

func verifySecret(secret *corev1.Secret) error {
	for _, token := range []string{_const.DynatracePaasToken, _const.DynatraceApiToken} {
		_, err := ExtractToken(secret, token)
		if err != nil {
			return fmt.Errorf("invalid secret %s, %s", secret.Name, err)
		}
	}

	return nil
}

func ExtractToken(secret *corev1.Secret, key string) (string, error) {
	value, ok := secret.Data[key]
	if !ok {
		err := fmt.Errorf("missing token %s", key)
		return "", err
	}

	return strings.TrimSpace(string(value)), nil
}

func GetTokensName(obj *dynatracev1alpha1.ActiveGate) string {
	if tkns := obj.Spec.Tokens; tkns != "" {
		return tkns
	}
	return obj.GetName()
}

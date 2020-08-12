package parser

import (
	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/controller/const"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"testing"
)

func TestNewTokens(t *testing.T) {
	secret := createTestSecret()
	tokens, err := NewTokens(secret)

	assert.Nil(t, err)
	assert.NotNil(t, tokens)

	assert.Equal(t, ApiToken, tokens.ApiToken)
	assert.Equal(t, PaasToken, tokens.PaasToken)
}

func TestNewTokens_NoApiToken(t *testing.T) {
	secret := createTestSecret()

	delete(secret.Data, _const.DynatraceApiToken)

	tokens, err := NewTokens(secret)

	assert.NotNil(t, err)
	assert.Nil(t, tokens)
}

func TestNewTokens_NoPaasToken(t *testing.T) {
	secret := createTestSecret()

	delete(secret.Data, _const.DynatracePaasToken)

	tokens, err := NewTokens(secret)

	assert.NotNil(t, err)
	assert.Nil(t, tokens)
}

func createTestSecret() *corev1.Secret {
	return &corev1.Secret{
		Data: map[string][]byte{
			_const.DynatraceApiToken:  []byte(ApiToken),
			_const.DynatracePaasToken: []byte(PaasToken),
		},
	}
}

const (
	ApiToken  = "ApiToken"
	PaasToken = "PaasToken"
)

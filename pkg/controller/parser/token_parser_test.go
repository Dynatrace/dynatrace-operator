package parser

import (
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"testing"
)

func TestNewTokens(t *testing.T) {
	secret := createTestSecret()
	tokens, err := NewTokens(secret)

	assert.Nil(t, err)
	assert.NotNil(t, tokens)

	assert.Equal(t, API_TOKEN, tokens.ApiToken)
	assert.Equal(t, PAAS_TOKEN, tokens.PaasToken)
}

func TestNewTokens_NoApiToken(t *testing.T) {
	secret := createTestSecret()

	delete(secret.Data, DynatraceApiToken)

	tokens, err := NewTokens(secret)

	assert.NotNil(t, err)
	assert.Nil(t, tokens)
}

func TestNewTokens_NoPaasToken(t *testing.T) {
	secret := createTestSecret()

	delete(secret.Data, DynatracePaasToken)

	tokens, err := NewTokens(secret)

	assert.NotNil(t, err)
	assert.Nil(t, tokens)
}

func createTestSecret() *corev1.Secret {
	return &corev1.Secret{
		Data: map[string][]byte{
			DynatraceApiToken:  []byte(API_TOKEN),
			DynatracePaasToken: []byte(PAAS_TOKEN),
		},
	}
}

const (
	EMPTY = ""

	API_TOKEN  = "ApiToken"
	PAAS_TOKEN = "PaasToken"
)

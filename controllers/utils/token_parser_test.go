package utils

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

const (
	testValue            = "test-value"
	testKey              = "test-key"
	testValueAlternative = "test-alternative-value"
)

func TestNewTokens(t *testing.T) {
	t.Run(`NewTokens extracts api and paas token from secret`, func(t *testing.T) {
		secret := corev1.Secret{
			Data: map[string][]byte{
				dtclient.DynatraceApiToken:  []byte(testValue),
				dtclient.DynatracePaasToken: []byte(testValueAlternative),
			}}
		tokens, err := NewTokens(&secret)

		assert.NoError(t, err)
		assert.NotNil(t, tokens)
		assert.Equal(t, testValue, tokens.ApiToken)
		assert.Equal(t, testValueAlternative, tokens.PaasToken)
	})
	t.Run(`NewTokens handles missing api or paas token`, func(t *testing.T) {
		secret := corev1.Secret{
			Data: map[string][]byte{
				dtclient.DynatraceApiToken: []byte(testValue),
			}}
		tokens, err := NewTokens(&secret)

		assert.Error(t, err)
		assert.Nil(t, tokens)
		assert.Contains(t, err.Error(), dtclient.DynatracePaasToken)

		secret = corev1.Secret{
			Data: map[string][]byte{
				dtclient.DynatracePaasToken: []byte(testValueAlternative),
			}}
		tokens, err = NewTokens(&secret)

		assert.Error(t, err)
		assert.Nil(t, tokens)
		assert.Contains(t, err.Error(), dtclient.DynatraceApiToken)

		secret = corev1.Secret{
			Data: map[string][]byte{}}
		tokens, err = NewTokens(&secret)

		assert.Error(t, err)
		assert.Nil(t, tokens)
		assert.Contains(t, err.Error(), dtclient.DynatraceApiToken)
	})
	t.Run(`NewTokens handles nil secret`, func(t *testing.T) {
		tokens, err := NewTokens(nil)

		assert.Error(t, err)
		assert.Nil(t, tokens)
	})
}

func TestExtractToken(t *testing.T) {
	t.Run(`ExtractToken returns value from secret`, func(t *testing.T) {
		secret := corev1.Secret{
			Data: map[string][]byte{
				testKey:  []byte(testValue),
				testName: []byte(testValueAlternative),
			}}

		value, err := ExtractToken(&secret, testKey)

		assert.NoError(t, err)
		assert.Equal(t, value, testValue)

		value, err = ExtractToken(&secret, testName)

		assert.NoError(t, err)
		assert.Equal(t, value, testValueAlternative)
	})
	t.Run(`ExtractToken handles missing key`, func(t *testing.T) {
		secret := corev1.Secret{
			Data: map[string][]byte{}}

		value, err := ExtractToken(&secret, testKey)

		assert.Error(t, err)
		assert.Empty(t, value)
	})
}

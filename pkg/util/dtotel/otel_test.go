package dtotel

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetConfig(t *testing.T) {
	const namespace = "dynatrace"

	const expectedEndpoint = "abc12345.dynatrace.com"

	const expectedApiToken = "dt01234.abcdef.abcdefg"

	t.Run("happy path", func(t *testing.T) {
		clt := fake.NewClient(&corev1.Secret{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dynatrace-operator-otel-config",
				Namespace: namespace,
			},
			Data: map[string][]byte{
				"endpoint": []byte(expectedEndpoint),
				"apiToken": []byte(expectedApiToken),
			},
		})

		endpoint, apiToken, err := getOtelConfig(context.Background(), clt, namespace)
		require.NoError(t, err)
		assert.Equal(t, expectedEndpoint, endpoint)
		assert.Equal(t, expectedApiToken, apiToken)
	})
	t.Run("no endpoint", func(t *testing.T) {
		clt := fake.NewClient(&corev1.Secret{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dynatrace-operator-otel-config",
				Namespace: namespace,
			},
			Data: map[string][]byte{
				"apiToken": []byte(expectedApiToken),
			},
		})

		endpoint, apiToken, err := getOtelConfig(context.Background(), clt, namespace)
		require.Error(t, err)
		assert.Equal(t, "", endpoint)
		assert.Equal(t, "", apiToken)
	})
	t.Run("no token", func(t *testing.T) {
		clt := fake.NewClient(&corev1.Secret{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dynatrace-operator-otel-config",
				Namespace: namespace,
			},
			Data: map[string][]byte{
				"endpoint": []byte(expectedEndpoint),
			},
		})

		endpoint, apiToken, err := getOtelConfig(context.Background(), clt, namespace)
		require.Error(t, err)
		assert.Equal(t, "", endpoint)
		assert.Equal(t, "", apiToken)
	})
	t.Run("no secret", func(t *testing.T) {
		clt := fake.NewClient()

		endpoint, apiToken, err := getOtelConfig(context.Background(), clt, namespace)
		require.Error(t, err)
		assert.Equal(t, "", endpoint)
		assert.Equal(t, "", apiToken)
	})
}

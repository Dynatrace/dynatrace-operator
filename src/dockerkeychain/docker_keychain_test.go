package dockerkeychain

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	errors2 "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	registryName = "docker.test.com"
	testToken    = "test-token"
	testPassword = "test-password"
	testAuth     = "dGVzdC10b2tlbjp0ZXN0LXBhc3N3b3Jk" // echo -n "test-token:test-password" | base64
	dockerConfig = "{\"auths\":{\"" + registryName + "\":{\"username\":\"" + testToken + "\",\"password\":\"" + testPassword + "\",\"auth\":\"" + testAuth + "\"}}}"
)

func TestNewDockerKeychain(t *testing.T) {
	t.Run("secret not found", func(t *testing.T) {
		pullSecret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-secret",
				Namespace: "dynatrace",
			},
		}
		client := fake.NewClient()

		keychain := NewDockerKeychain()
		err := keychain.LoadDockerConfigFromSecret(context.TODO(), client, pullSecret)
		require.Error(t, err)
		require.True(t, errors2.IsNotFound(err))
	})

	t.Run("invalid format of docker secret dockerconfigjson", func(t *testing.T) {
		pullSecret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-secret",
				Namespace: "dynatrace",
			},
			Data: map[string][]byte{
				".dockerconfigjson": []byte("invalid format"),
			},
		}
		client := fake.NewClientWithIndex(&pullSecret)

		keychain := NewDockerKeychain()
		err := keychain.LoadDockerConfigFromSecret(context.TODO(), client, pullSecret)
		require.Error(t, err)

		var syntaxError *json.SyntaxError
		ok := errors.As(err, &syntaxError)
		require.True(t, ok)
		assert.Equal(t, int64(1), syntaxError.Offset)
	})

	t.Run("valid config provided", func(t *testing.T) {
		pullSecret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-secret",
				Namespace: "dynatrace",
			},
			Data: map[string][]byte{
				corev1.DockerConfigJsonKey: []byte(dockerConfig),
			},
			Type: corev1.SecretTypeDockerConfigJson,
		}
		client := fake.NewClientWithIndex(&pullSecret)

		keychain := NewDockerKeychain()
		err := keychain.LoadDockerConfigFromSecret(context.TODO(), client, pullSecret)
		require.NoError(t, err)
		registry, err := name.NewRegistry(registryName, name.StrictValidation)
		require.NoError(t, err)

		authenticator, err := keychain.Resolve(registry)

		require.NoError(t, err)
		assert.NotNil(t, authenticator)
		auth, err := authenticator.Authorization()
		require.NoError(t, err)
		assert.Equal(t, testToken, auth.Username)
		assert.Equal(t, testPassword, auth.Password)
	})
}

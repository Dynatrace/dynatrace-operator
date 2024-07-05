package dockerkeychain

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	registryName         = "docker.test.com"
	registryTestToken    = "test-token"
	registryTestPassword = "test-password"
	registryTestAuth     = "dGVzdC10b2tlbjp0ZXN0LXBhc3N3b3Jk" // echo -n "test-token:test-password" | base64
	registryDockerConfig = "{\"auths\":{\"" + registryName + "\":{\"username\":\"" + registryTestToken + "\",\"password\":\"" + registryTestPassword + "\",\"auth\":\"" + registryTestAuth + "\"}}}"

	registryCustomTestToken    = "custom-test-token"
	registryCustomTestPassword = "custom-test-password"
	registryCustomTestAuth     = "Y3VzdG9tLXRlc3QtdG9rZW46Y3VzdG9tLXRlc3QtcGFzc3dvcmQ=" // echo -n "custom-test-token:custom-test-password" | base64
	registryCustomDockerConfig = "{\"auths\":{\"" + registryName + "\":{\"username\":\"" + registryCustomTestToken + "\",\"password\":\"" + registryCustomTestPassword + "\",\"auth\":\"" + registryCustomTestAuth + "\"}}}"

	e2eRegistryName         = "e2e.test.com"
	e2eRegistryTestToken    = "e2e-test-token"
	e2eRegistryTestPassword = "e2e-test-password"
	e2eRegistryTestAuth     = "ZTJlLXRlc3QtdG9rZW46ZTJlLXRlc3QtcGFzc3dvcmQ=" // echo -n "e2e-test-token:e2e-test-password" | base64
	e2eRegistryDockerConfig = "{\"auths\":{\"" + e2eRegistryName + "\":{\"username\":\"" + e2eRegistryTestToken + "\",\"password\":\"" + e2eRegistryTestPassword + "\",\"auth\":\"" + e2eRegistryTestAuth + "\"}}}"

	tenantPullSecretName = "dynakube-pull-secret"
	customPullSecretName = "custom-pull-secret"
)

func TestNewDockerKeychain(t *testing.T) {
	t.Run("secret not found, try without secret", func(t *testing.T) {
		pullSecret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-secret",
				Namespace: "dynatrace",
			},
		}
		client := fake.NewClient()

		_, err := NewDockerKeychain(context.TODO(), client, pullSecret)
		require.NoError(t, err)
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

		_, err := NewDockerKeychain(context.TODO(), client, pullSecret)
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
				corev1.DockerConfigJsonKey: []byte(registryDockerConfig),
			},
			Type: corev1.SecretTypeDockerConfigJson,
		}
		client := fake.NewClientWithIndex(&pullSecret)

		keychain, err := NewDockerKeychain(context.TODO(), client, pullSecret)
		require.NoError(t, err)
		registry, err := name.NewRegistry(registryName, name.StrictValidation)
		require.NoError(t, err)

		authenticator, err := keychain.Resolve(registry)

		require.NoError(t, err)
		assert.NotNil(t, authenticator)
		auth, err := authenticator.Authorization()
		require.NoError(t, err)
		assert.Equal(t, registryTestToken, auth.Username)
		assert.Equal(t, registryTestPassword, auth.Password)
	})
}

func TestNewDockerKeychains(t *testing.T) {
	t.Run("tenant secret not found", func(t *testing.T) {
		client := fake.NewClient()

		_, err := NewDockerKeychains(context.TODO(), client, "dynatrace", []string{"dynakube-pull-secret"})
		require.NoError(t, err)
	})

	t.Run("the same registry", func(t *testing.T) {
		tenantPullSecret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      tenantPullSecretName,
				Namespace: "dynatrace",
			},
			Data: map[string][]byte{
				corev1.DockerConfigJsonKey: []byte(registryDockerConfig),
			},
			Type: corev1.SecretTypeDockerConfigJson,
		}
		customPullSecret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      customPullSecretName,
				Namespace: "dynatrace",
			},
			Data: map[string][]byte{
				corev1.DockerConfigJsonKey: []byte(registryCustomDockerConfig),
			},
			Type: corev1.SecretTypeDockerConfigJson,
		}
		client := fake.NewClientWithIndex(&tenantPullSecret, &customPullSecret)

		keychain, err := NewDockerKeychains(context.TODO(), client, "dynatrace", []string{tenantPullSecretName, customPullSecretName})
		require.NoError(t, err)
		registry, err := name.NewRegistry(registryName, name.StrictValidation)
		require.NoError(t, err)

		authenticator, err := keychain.Resolve(registry)

		require.NoError(t, err)
		assert.NotNil(t, authenticator)
		auth, err := authenticator.Authorization()
		require.NoError(t, err)
		assert.Equal(t, registryCustomTestToken, auth.Username)
		assert.Equal(t, registryCustomTestPassword, auth.Password)
	})

	t.Run("different registries", func(t *testing.T) {
		tenantPullSecret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      tenantPullSecretName,
				Namespace: "dynatrace",
			},
			Data: map[string][]byte{
				corev1.DockerConfigJsonKey: []byte(registryDockerConfig),
			},
			Type: corev1.SecretTypeDockerConfigJson,
		}
		customPullSecret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      customPullSecretName,
				Namespace: "dynatrace",
			},
			Data: map[string][]byte{
				corev1.DockerConfigJsonKey: []byte(e2eRegistryDockerConfig),
			},
			Type: corev1.SecretTypeDockerConfigJson,
		}
		client := fake.NewClientWithIndex(&tenantPullSecret, &customPullSecret)

		keychain, err := NewDockerKeychains(context.TODO(), client, "dynatrace", []string{tenantPullSecretName, customPullSecretName})
		require.NoError(t, err)

		registry, err := name.NewRegistry(registryName, name.StrictValidation)
		require.NoError(t, err)
		authenticator, err := keychain.Resolve(registry)
		require.NoError(t, err)
		assert.NotNil(t, authenticator)
		auth, err := authenticator.Authorization()
		require.NoError(t, err)
		assert.Equal(t, registryTestToken, auth.Username)
		assert.Equal(t, registryTestPassword, auth.Password)

		registry, err = name.NewRegistry(e2eRegistryName, name.StrictValidation)
		require.NoError(t, err)
		authenticator, err = keychain.Resolve(registry)
		require.NoError(t, err)
		assert.NotNil(t, authenticator)
		auth, err = authenticator.Authorization()
		require.NoError(t, err)
		assert.Equal(t, e2eRegistryTestToken, auth.Username)
		assert.Equal(t, e2eRegistryTestPassword, auth.Password)
	})
}

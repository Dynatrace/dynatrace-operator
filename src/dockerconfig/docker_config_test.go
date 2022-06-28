package dockerconfig

import (
	"context"
	"fmt"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testName  = "test-name"
	testKey   = "testKey"
	testValue = "testValue"
)

func TestNewDockerConfig(t *testing.T) {
	apiReader := fake.NewClient()
	t.Run("empty dynakube", func(t *testing.T) {
		dynakube := dynatracev1beta1.DynaKube{}
		dockerConfig := NewDockerConfig(apiReader, dynakube)

		require.NotNil(t, dockerConfig)
		assert.NotNil(t, dockerConfig.Auths)
		assert.Empty(t, dockerConfig.Auths)
		assert.Equal(t, apiReader, dockerConfig.ApiReader)
		assert.False(t, dockerConfig.SkipCertCheck)
	})
	t.Run("empty skipCertCheck", func(t *testing.T) {
		dynakube := dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				SkipCertCheck: true,
			},
		}
		dockerConfig := NewDockerConfig(apiReader, dynakube)

		require.NotNil(t, dockerConfig)
		assert.NotNil(t, dockerConfig.Auths)
		assert.Empty(t, dockerConfig.Auths)
		assert.Equal(t, apiReader, dockerConfig.ApiReader)
		assert.True(t, dockerConfig.SkipCertCheck)
	})
}

func TestSetupAuths(t *testing.T) {
	t.Run("using default pull secret", func(t *testing.T) {
		dynakube := dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name: testName,
			},
		}
		pullSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: dynakube.PullSecret(),
			},
			Data: map[string][]byte{
				".dockerconfigjson": []byte(
					fmt.Sprintf(`{ "auths": { "%s": { "username": "%s", "password": "%s" } } }`, testKey, testName, testValue)),
			},
		}
		apiReader := fake.NewClient(pullSecret)
		dockerConfig := NewDockerConfig(apiReader, dynakube)

		err := dockerConfig.SetupAuths(context.TODO())

		require.NoError(t, err)
		assert.NotNil(t, dockerConfig.Auths)
		assert.NotEmpty(t, dockerConfig.Auths)
		assert.Equal(t, testName, dockerConfig.Auths[testKey].Username)
		assert.Equal(t, testValue, dockerConfig.Auths[testKey].Password)

	})
	t.Run("using custom pull secret", func(t *testing.T) {
		dynakube := dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				CustomPullSecret: testName,
			},
		}
		pullSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: dynakube.PullSecret(),
			},
			Data: map[string][]byte{
				".dockerconfigjson": []byte(
					fmt.Sprintf(`{ "auths": { "%s": { "username": "%s", "password": "%s" } } }`, testKey, testName, testValue)),
			},
		}
		apiReader := fake.NewClient(pullSecret)
		dockerConfig := NewDockerConfig(apiReader, dynakube)

		err := dockerConfig.SetupAuths(context.TODO())

		require.NoError(t, err)
		assert.NotNil(t, dockerConfig.Auths)
		assert.NotEmpty(t, dockerConfig.Auths)
		assert.Equal(t, testName, dockerConfig.Auths[testKey].Username)
		assert.Equal(t, testValue, dockerConfig.Auths[testKey].Password)
	})
	t.Run("handles invalid json", func(t *testing.T) {
		dynakube := dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				CustomPullSecret: testName,
			},
		}
		pullSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: dynakube.PullSecret(),
			},
			Data: map[string][]byte{
				".dockerconfigjson": []byte("asd"),
			},
		}
		apiReader := fake.NewClient(pullSecret)
		dockerConfig := NewDockerConfig(apiReader, dynakube)

		err := dockerConfig.SetupAuths(context.TODO())

		require.Error(t, err)
		assert.Empty(t, dockerConfig.Auths)
	})
	t.Run("handles no secret", func(t *testing.T) {
		dynakube := dynatracev1beta1.DynaKube{}

		apiReader := fake.NewClient()
		dockerConfig := NewDockerConfig(apiReader, dynakube)

		err := dockerConfig.SetupAuths(context.TODO())

		require.Error(t, err)
		assert.Empty(t, dockerConfig.Auths)
	})

}

func TestParseDockerAuthsFromSecret(t *testing.T) {
	t.Run("parseDockerAuthsFromSecret handles nil secret", func(t *testing.T) {
		auths, err := parseDockerAuthsFromSecret(nil)
		require.Nil(t, auths)
		require.Error(t, err)
	})
	t.Run("parseDockerAuthsFromSecret handles missing secret data", func(t *testing.T) {
		auths, err := parseDockerAuthsFromSecret(&corev1.Secret{})
		require.Nil(t, auths)
		require.Error(t, err)
	})
	t.Run("parseDockerAuthsFromSecret handles invalid json", func(t *testing.T) {
		auths, err := parseDockerAuthsFromSecret(&corev1.Secret{
			Data: map[string][]byte{
				".dockerconfigjson": []byte(`invalid json`),
			},
		})

		require.Nil(t, auths)
		require.Error(t, err)
	})
	t.Run("parseDockerAuthsFromSecret handles valid json", func(t *testing.T) {
		auths, err := parseDockerAuthsFromSecret(&corev1.Secret{
			Data: map[string][]byte{
				".dockerconfigjson": []byte(
					fmt.Sprintf(`{ "auths": { "%s": { "username": "%s", "password": "%s" } } }`, testKey, testName, testValue)),
			},
		})

		require.NoError(t, err)
		require.NotEmpty(t, auths)
		assert.Contains(t, auths, testKey)
		assert.Equal(t, testName, auths[testKey].Username)
		assert.Equal(t, testValue, auths[testKey].Password)
	})
}

func TestSaveCustomCAs(t *testing.T) {
	caSecretName := "ca-secret"
	namespace := "test-namespace"
	testPath := "/test/path"

	dynakube := dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dk",
			Namespace: namespace,
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			TrustedCAs: caSecretName,
		},
	}

	t.Run("fail because of bad secret", func(t *testing.T) {
		client := fake.NewClient(&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      caSecretName,
				Namespace: namespace,
			},
			Data: map[string][]byte{
				"invalid-key": []byte(`invalid json`),
			},
		})
		dockerConfig := DockerConfig{
			ApiReader: client,
			Dynakube:  &dynakube,
		}
		fs := afero.Afero{Fs: afero.NewMemMapFs()}
		err := dockerConfig.SaveCustomCAs(context.TODO(), fs, testPath)
		require.Error(t, err)
	})

	t.Run("stores it in the given fs", func(t *testing.T) {
		client := fake.NewClient(&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      caSecretName,
				Namespace: namespace,
			},
			Data: map[string]string{
				dtclient.CustomCertificatesConfigMapKey: `I-am-a-cert-trust-me`,
			},
		})
		dockerConfig := DockerConfig{
			ApiReader: client,
			Dynakube:  &dynakube,
		}
		fs := afero.Afero{Fs: afero.NewMemMapFs()}
		err := dockerConfig.SaveCustomCAs(context.TODO(), fs, testPath)
		require.NoError(t, err)
		exists, err := fs.Exists(testPath)
		require.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, testPath, dockerConfig.TrustedCertsPath)
	})
}

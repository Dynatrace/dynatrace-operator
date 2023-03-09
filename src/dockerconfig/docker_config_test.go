package dockerconfig

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
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
		assert.NotEqual(t, "", dockerConfig.RegistryAuthPath)
		assert.Equal(t, "", dockerConfig.TrustedCertsPath)
		assert.Equal(t, apiReader, dockerConfig.ApiReader)
		assert.False(t, dockerConfig.SkipCertCheck())
	})
	t.Run("empty skipCertCheck", func(t *testing.T) {
		dynakube := dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				SkipCertCheck: true,
			},
		}
		dockerConfig := NewDockerConfig(apiReader, dynakube)

		require.NotNil(t, dockerConfig)
		assert.NotEqual(t, "", dockerConfig.RegistryAuthPath)
		assert.Equal(t, "", dockerConfig.TrustedCertsPath)
		assert.Equal(t, apiReader, dockerConfig.ApiReader)
		assert.True(t, dockerConfig.SkipCertCheck())
	})
	t.Run("regular dynakube", func(t *testing.T) {
		dynakube := dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testName,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				TrustedCAs: "secure-cert-here",
			},
		}
		dockerConfig := NewDockerConfig(apiReader, dynakube)

		require.NotNil(t, dockerConfig)
		assert.Equal(t, path.Join(TmpPath, RegistryAuthDir, dynakube.Name), dockerConfig.RegistryAuthPath)
		assert.Equal(t, path.Join(TmpPath, CADir, dynakube.Name), dockerConfig.TrustedCertsPath)
		assert.Equal(t, apiReader, dockerConfig.ApiReader)
		assert.False(t, dockerConfig.SkipCertCheck())
	})
}

func checkFileContents(t *testing.T, fs afero.Afero, targetPath, expectedContent string) {
	targetFile, err := fs.Open(targetPath)
	assert.NoError(t, err)
	targetFileContent, err := io.ReadAll(targetFile)
	assert.NoError(t, err)
	assert.Equal(t, expectedContent, string(targetFileContent))
}

func TestSetupAuths(t *testing.T) {
	t.Run("using default pull secret", func(t *testing.T) {
		registryAuthContent := fmt.Sprintf(`{ "auths": { "%s": { "username": "%s", "password": "%s" } } }`, testKey, testName, testValue)

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
				".dockerconfigjson": []byte(registryAuthContent),
			},
		}
		apiReader := fake.NewClient(pullSecret)
		dockerConfig := NewDockerConfig(apiReader, dynakube)

		fs := afero.Afero{Fs: afero.NewBasePathFs(afero.NewOsFs(), path.Join(os.TempDir(), "dttest"))}
		defer func(fs afero.Afero, path string) {
			_ = fs.RemoveAll(path)
		}(fs, "/")

		err := dockerConfig.StoreRequiredFiles(context.TODO(), fs)
		require.NoError(t, err)

		checkFileContents(t, fs, dockerConfig.RegistryAuthPath, registryAuthContent)
	})
	t.Run("using default pull secret with ca certs set", func(t *testing.T) {
		registryAuthContent := fmt.Sprintf(`{ "auths": { "%s": { "username": "%s", "password": "%s" } } }`, testKey, testName, testValue)
		secureCertName := "secure-cert-name"

		dynakube := dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name: testName,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				TrustedCAs: secureCertName,
			},
		}
		pullSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: dynakube.PullSecret(),
			},
			Data: map[string][]byte{
				".dockerconfigjson": []byte(registryAuthContent),
			},
		}
		caCertsConfigMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: secureCertName,
			},
			Data: map[string]string{
				dtclient.CustomCertificatesConfigMapKey: testValue,
			},
		}
		apiReader := fake.NewClient(pullSecret, caCertsConfigMap)
		dockerConfig := NewDockerConfig(apiReader, dynakube)

		fs := afero.Afero{Fs: afero.NewBasePathFs(afero.NewOsFs(), path.Join(os.TempDir(), "dttest"))}
		defer func(fs afero.Afero, path string) {
			_ = fs.RemoveAll(path)
		}(fs, "/")

		err := dockerConfig.StoreRequiredFiles(context.TODO(), fs)
		require.NoError(t, err)

		checkFileContents(t, fs, dockerConfig.RegistryAuthPath, registryAuthContent)
		checkFileContents(t, fs, dockerConfig.TrustedCertsPath, testValue)
	})
	t.Run("using custom pull secret", func(t *testing.T) {
		registryAuthContent := fmt.Sprintf(`{ "auths": { "%s": { "username": "%s", "password": "%s" } } }`, testKey, testName, testValue)

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
				".dockerconfigjson": []byte(registryAuthContent),
			},
		}
		apiReader := fake.NewClient(pullSecret)
		dockerConfig := NewDockerConfig(apiReader, dynakube)

		fs := afero.Afero{Fs: afero.NewBasePathFs(afero.NewOsFs(), path.Join(os.TempDir(), "dttest"))}
		defer func(fs afero.Afero, path string) {
			_ = fs.RemoveAll(path)
		}(fs, "/")

		err := dockerConfig.StoreRequiredFiles(context.TODO(), fs)
		require.NoError(t, err)

		checkFileContents(t, fs, dockerConfig.RegistryAuthPath, registryAuthContent)
	})
	t.Run("using preset pull secret", func(t *testing.T) {
		registryAuthContent := fmt.Sprintf(`{ "auths": { "%s": { "username": "%s", "password": "%s" } } }`, testKey, testName, testValue)

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
				".dockerconfigjson": []byte(registryAuthContent),
			},
		}
		apiReader := fake.NewClient()
		dockerConfig := NewDockerConfig(apiReader, dynakube)
		dockerConfig.SetRegistryAuthSecret(pullSecret)

		fs := afero.Afero{Fs: afero.NewBasePathFs(afero.NewOsFs(), path.Join(os.TempDir(), "dttest"))}
		defer func(fs afero.Afero, path string) {
			_ = fs.RemoveAll(path)
		}(fs, "/")

		err := dockerConfig.StoreRequiredFiles(context.TODO(), fs)
		require.NoError(t, err)

		checkFileContents(t, fs, dockerConfig.RegistryAuthPath, registryAuthContent)
	})
	t.Run("handles no secret", func(t *testing.T) {
		dynakube := dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name: testName,
			},
		}

		apiReader := fake.NewClient()
		dockerConfig := NewDockerConfig(apiReader, dynakube)

		fs := afero.Afero{Fs: afero.NewBasePathFs(afero.NewOsFs(), path.Join(os.TempDir(), "dttest"))}
		defer func(fs afero.Afero, path string) {
			_ = fs.RemoveAll(path)
		}(fs, "/")
		err := dockerConfig.StoreRequiredFiles(context.TODO(), fs)

		require.Error(t, err)
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
	t.Run("parseDockerAuthsFromSecret handles valid json", func(t *testing.T) {
		auths, err := parseDockerAuthsFromSecret(&corev1.Secret{
			Data: map[string][]byte{
				".dockerconfigjson": []byte(
					fmt.Sprintf(`{ "auths": { "%s": { "username": "%s", "password": "%s" } } }`, testKey, testName, testValue)),
			},
		})

		require.NoError(t, err)
		require.NotEmpty(t, auths)
		assert.Contains(t, string(auths), testKey)
	})
}

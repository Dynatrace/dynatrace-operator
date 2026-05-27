//go:build e2e

package tenant

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/e2e/project"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

var (
	defaultSingleTenant      = filepath.Join(project.TestDataDir(), "secrets/single-tenant.yaml")
	defaultMultiTenant       = filepath.Join(project.TestDataDir(), "secrets/multi-tenant.yaml")
	defaultEdgeConnectTenant = filepath.Join(project.TestDataDir(), "secrets/edgeconnect-tenant.yaml")
)

type Secrets struct {
	Tenants []Secret `yaml:"tenants"`
}

type Secret struct {
	TenantUID          string `yaml:"tenantUid"`
	APIURL             string `yaml:"apiUrl"`
	APIToken           string `yaml:"apiToken"`
	DataIngestToken    string `yaml:"dataIngestToken"`
	APITokenNoSettings string `yaml:"apiTokenNoSettings"`
	PlatformToken      string `yaml:"platformToken"`
}

type Tokens struct {
	APIToken        string
	DataIngestToken string
}

type EdgeConnectSecret struct {
	TenantUID         string `yaml:"tenantUid"`
	Name              string `yaml:"name"`
	APIServer         string `yaml:"apiServer"`
	OauthClientID     string `yaml:"oAuthClientId"`
	OauthClientSecret string `yaml:"oAuthClientSecret"`
	Resource          string `yaml:"resource"`
}

func (s Secret) TokensWithSettingsScope() Tokens {
	return s.tokens(true)
}

func (s Secret) TokensWithoutSettingsScope() Tokens {
	return s.tokens(false)
}

func (s Secret) tokens(hasSettingsScope bool) Tokens {
	if WithPlatformToken() {
		return Tokens{APIToken: s.PlatformToken, DataIngestToken: s.DataIngestToken}
	}

	t := s.APIToken
	if !hasSettingsScope {
		t = s.APITokenNoSettings
	}

	return Tokens{APIToken: t, DataIngestToken: s.DataIngestToken}
}

func (s Secret) PlatformTokens() Tokens {
	return Tokens{APIToken: s.PlatformToken, DataIngestToken: s.DataIngestToken}
}

func (s Secret) ClassicTokens() Tokens {
	return Tokens{APIToken: s.APIToken, DataIngestToken: s.DataIngestToken}
}

func (t Tokens) IsEmpty() bool {
	return t.APIToken == "" && t.DataIngestToken == ""
}

func manyFromConfig(path string) ([]Secret, error) {
	secretConfigFile, err := os.ReadFile(path)

	if err != nil {
		return []Secret{}, errors.WithStack(err)
	}

	var result Secrets
	err = yaml.Unmarshal(secretConfigFile, &result)

	return result.Tenants, errors.WithStack(err)
}

func newFromConfig(path string) (Secret, error) {
	secretConfigFile, err := os.ReadFile(path)

	if err != nil {
		return Secret{}, errors.WithStack(err)
	}

	var result Secret
	err = yaml.Unmarshal(secretConfigFile, &result)

	return result, errors.WithStack(err)
}

func GetSingleTenantSecret(t *testing.T) Secret {
	secret, err := newFromConfig(defaultSingleTenant)
	if err != nil {
		t.Fatal("Couldn't read tenant secret from filesystem", err)
	}

	return secret
}

func GetMultiTenantSecret(t *testing.T) []Secret {
	secrets, err := manyFromConfig(defaultMultiTenant)
	if err != nil {
		t.Fatal("Couldn't read tenant secret from filesystem", err)
	}

	return secrets
}

func GetEdgeConnectTenantSecret(t *testing.T) EdgeConnectSecret {
	secretConfigFile, err := os.ReadFile(defaultEdgeConnectTenant)

	if err != nil {
		t.Fatal("Couldn't read edgeconnect tenant secret from filesystem", err)
	}

	var result EdgeConnectSecret
	err = yaml.Unmarshal(secretConfigFile, &result)

	if err != nil {
		t.Fatal("Couldn't unmarshal edgeconnect tenant secret from file", err)
	}

	return result
}

// WithPlatformToken reports whether the test run is configured to use a platform token
// instead of a classic API token (controlled by the PLATFORM_TOKEN=true env var).
func WithPlatformToken() bool {
	return os.Getenv("PLATFORM_TOKEN") == "true"
}

func CreateTenantSecret(tokens Tokens, name, namespace string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		defaultSecret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Labels: map[string]string{
					"type": "tenant",
				},
			},
			Data: map[string][]byte{
				"apiToken": []byte(tokens.APIToken),
			},
		}

		if tokens.DataIngestToken != "" {
			defaultSecret.Data["dataIngestToken"] = []byte(tokens.DataIngestToken)
		}

		err := envConfig.Client().Resources().Create(ctx, &defaultSecret)

		if k8serrors.IsAlreadyExists(err) {
			require.NoError(t, envConfig.Client().Resources().Update(ctx, &defaultSecret))

			return ctx
		}

		require.NoError(t, err)

		return ctx
	}
}

func DeleteTenantSecret(secretName, secretNamespace string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		secret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: secretNamespace,
			},
		}
		err := envConfig.Client().Resources().Delete(ctx, &secret)
		if err != nil {
			if k8serrors.IsNotFound(err) {
				err = nil
			}
		}
		require.NoError(t, err)

		return ctx
	}
}

func CreateClientSecret(secretConfig *EdgeConnectSecret, name, namespace string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		defaultSecret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Data: map[string][]byte{
				"oauth-client-id":     []byte(secretConfig.OauthClientID),
				"oauth-client-secret": []byte(secretConfig.OauthClientSecret),
			},
		}

		err := envConfig.Client().Resources().Create(ctx, &defaultSecret)

		if k8serrors.IsAlreadyExists(err) {
			require.NoError(t, envConfig.Client().Resources().Update(ctx, &defaultSecret))

			return ctx
		}

		require.NoError(t, err)

		return ctx
	}
}

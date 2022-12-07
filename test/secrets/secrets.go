package secrets

import (
	"context"
	"path"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/project"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

var (
	defaultSingleTenant = path.Join(project.TestDataDir(), "secrets/single-tenant.yaml")
	defaultMultiTenant  = path.Join(project.TestDataDir(), "secrets/multi-tenant.yaml")
)

type Secrets struct {
	Tenants []Secret `yaml:"tenants"`
}

type Secret struct {
	TenantUid string `yaml:"tenantUid"`
	ApiUrl    string `yaml:"apiUrl"`
	ApiToken  string `yaml:"apiToken"`
}

func ManyFromConfig(fs afero.Fs, path string) ([]Secret, error) {
	secretConfigFile, err := afero.ReadFile(fs, path)

	if err != nil {
		return []Secret{}, errors.WithStack(err)
	}

	var result Secrets
	err = yaml.Unmarshal(secretConfigFile, &result)

	return result.Tenants, errors.WithStack(err)
}

func NewFromConfig(fs afero.Fs, path string) (Secret, error) {
	secretConfigFile, err := afero.ReadFile(fs, path)

	if err != nil {
		return Secret{}, errors.WithStack(err)
	}

	var result Secret
	err = yaml.Unmarshal(secretConfigFile, &result)

	return result, errors.WithStack(err)
}

func DefaultSingleTenant(fs afero.Fs) (Secret, error) {
	return NewFromConfig(fs, defaultSingleTenant)
}

func DefaultMultiTenant(fs afero.Fs) ([]Secret, error) {
	return ManyFromConfig(fs, defaultMultiTenant)
}

func ApplyDefault(secretConfig Secret) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		defaultSecret := v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dynakube",
				Namespace: "dynatrace",
			},
			Data: map[string][]byte{
				"apiToken": []byte(secretConfig.ApiToken),
			},
		}

		require.NoError(t, environmentConfig.Client().Resources().Create(ctx, &defaultSecret))

		return ctx
	}
}

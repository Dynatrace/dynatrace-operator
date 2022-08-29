package secrets

import (
	"context"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"testing"
)

type Secrets struct {
	ApiUrl   string `yaml:"apiUrl"`
	ApiToken string `yaml:"apiToken"`
}

func NewFromConfig(fs afero.Fs, path string) (Secrets, error) {
	secretConfigFile, err := afero.ReadFile(fs, path)

	if err != nil {
		return Secrets{}, errors.WithStack(err)
	}

	var result Secrets
	err = yaml.Unmarshal(secretConfigFile, &result)

	return result, errors.WithStack(err)
}

func ApplyDefault(secretConfig Secrets) features.Func {
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

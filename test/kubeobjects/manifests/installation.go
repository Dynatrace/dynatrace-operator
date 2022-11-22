package manifests

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func InstallFromFile(path string) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		kubernetesManifest, err := os.Open(path)
		defer func() { require.NoError(t, kubernetesManifest.Close()) }()
		require.NoError(t, err)

		resources := environmentConfig.Client().Resources()
		require.NoError(t, decoder.DecodeEach(ctx, kubernetesManifest, decoder.IgnoreErrorHandler(decoder.CreateHandler(resources), k8serrors.IsAlreadyExists)))

		return ctx
	}
}

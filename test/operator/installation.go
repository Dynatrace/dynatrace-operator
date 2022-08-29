package operator

import (
	"context"
	"github.com/Dynatrace/dynatrace-operator/test/deployment"
	"github.com/stretchr/testify/require"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"os"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"testing"
)

func InstallForKubernetes(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
	kubernetesManifest, err := os.Open("../config/deploy/kubernetes/kubernetes-all.yaml")
	// For some reason 'Close()' is marked as an unresolved reference in IntelliJ
	// It does work and compile as expected when run though
	defer func() { require.NoError(t, kubernetesManifest.Close()) }()
	require.NoError(t, err)

	resources := environmentConfig.Client().Resources()
	require.NoError(t, decoder.DecodeEach(ctx, kubernetesManifest, decoder.IgnoreErrorHandler(decoder.CreateHandler(resources), func(err error) bool {
		// Ignore if the resource already exists
		return k8serrors.IsAlreadyExists(err)
	})))

	return ctx
}

func WaitForDeployment() features.Func {
	return deployment.WaitFor("dynatrace-operator", "dynatrace")
}

package operator

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
)

func Get(t *testing.T, ctx context.Context, resource *resources.Resources) (pods corev1.PodList) {
	require.NoError(t, resource.WithNamespace("dynatrace").List(ctx, &pods))
	return pods
}

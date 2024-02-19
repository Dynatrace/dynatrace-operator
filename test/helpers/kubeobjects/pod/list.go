//go:build e2e

package pod

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
)

func List(t *testing.T, ctx context.Context, resource *resources.Resources, namespaceName string) corev1.PodList {
	var pods corev1.PodList

	require.NoError(t, resource.WithNamespace(namespaceName).List(ctx, &pods))

	return pods
}

//go:build e2e

package k8spod

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
)

func List(ctx context.Context, t *testing.T, resource *resources.Resources, namespace string) corev1.PodList {
	var pods corev1.PodList

	require.NoError(t, resource.WithNamespace(namespace).List(ctx, &pods))

	return pods
}

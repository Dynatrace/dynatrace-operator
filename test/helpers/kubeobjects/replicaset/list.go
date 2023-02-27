//go:build e2e

package replicaset

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
)

func List(t *testing.T, ctx context.Context, resource *resources.Resources, namespaceName string) appsv1.ReplicaSetList {
	var replicasets appsv1.ReplicaSetList

	require.NoError(t, resource.WithNamespace(namespaceName).List(ctx, &replicasets))
	return replicasets
}

//go:build e2e

package k8sreplicaset

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
)

func List(t *testing.T, ctx context.Context, resource *resources.Resources, namespaceName string) appsv1.ReplicaSetList {
	var replicasets appsv1.ReplicaSetList

	require.NoError(t, resource.WithNamespace(namespaceName).List(ctx, &replicasets))

	return replicasets
}

func GetForOwner(ctx context.Context, t *testing.T, resource *resources.Resources, ownerName, namespace string) *appsv1.ReplicaSet {
	replicasets := listForNamespace(ctx, t, resource, namespace)

	for _, replicaset := range replicasets.Items {
		if len(replicaset.OwnerReferences) < 1 {
			continue
		}

		if replicaset.OwnerReferences[0].Name == ownerName {
			return &replicaset
		}
	}

	return nil
}

func listForNamespace(ctx context.Context, t *testing.T, resource *resources.Resources, namespace string) appsv1.ReplicaSetList {
	var replicasets appsv1.ReplicaSetList
	err := resource.WithNamespace(namespace).List(ctx, &replicasets)

	if err != nil {
		if k8serrors.IsNotFound(err) {
			err = nil
		}
		require.NoError(t, err)
	}

	return replicasets
}

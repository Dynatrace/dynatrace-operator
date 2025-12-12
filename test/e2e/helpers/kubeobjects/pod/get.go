//go:build e2e

package pod

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
)

func GetPodsForOwner(ctx context.Context, t *testing.T, resource *resources.Resources, ownerName, namespace string) corev1.PodList {
	pods := GetPodsForNamespace(ctx, t, resource, namespace)

	var targetPods corev1.PodList
	for _, pod := range pods.Items {
		if len(pod.OwnerReferences) < 1 {
			continue
		}

		if pod.OwnerReferences[0].Name == ownerName {
			targetPods.Items = append(targetPods.Items, pod)
		}
	}

	return targetPods
}

func GetPodsForNamespace(ctx context.Context, t *testing.T, resource *resources.Resources, namespace string) corev1.PodList {
	var pods corev1.PodList
	err := resource.WithNamespace(namespace).List(ctx, &pods)

	if err != nil {
		if k8serrors.IsNotFound(err) {
			err = nil
		}
		require.NoError(t, err)
	}

	return pods
}

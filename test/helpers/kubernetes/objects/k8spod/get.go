//go:build e2e

package k8spod

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
)

func ListForOwner(ctx context.Context, t *testing.T, resource *resources.Resources, ownerName, namespace string) corev1.PodList {
	pods := List(ctx, t, resource, namespace)

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

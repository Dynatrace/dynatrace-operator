//go:build e2e

package daemonset

import (
	"golang.org/x/net/context"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
)

func ForEachPod(ctx context.Context, resource *resources.Resources, daemonsetName string, daemonsetNamespace string, actionFunc PodConsumer) error {
	return NewQuery(ctx, resource, client.ObjectKey{
		Name:      daemonsetName,
		Namespace: daemonsetNamespace,
	}).ForEachPod(actionFunc)
}

package csi

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/daemonset"
	"github.com/Dynatrace/dynatrace-operator/test/operator"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
)

const (
	Name      = "dynatrace-oneagent-csi-driver"
	Namespace = operator.Namespace
)

func Get(ctx context.Context, resource *resources.Resources) (appsv1.DaemonSet, error) {
	return daemonset.NewQuery(ctx, resource, client.ObjectKey{
		Name:      Name,
		Namespace: Namespace,
	}).Get()
}

func ForEachPod(ctx context.Context, resource *resources.Resources, consumer daemonset.PodConsumer) error {
	return daemonset.NewQuery(ctx, resource, client.ObjectKey{
		Name:      Name,
		Namespace: Namespace,
	}).ForEachPod(consumer)
}

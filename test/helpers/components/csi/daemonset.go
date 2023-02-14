//go:build e2e

package csi

import (
	"bytes"
	"context"
	"testing"

	dtcsi "github.com/Dynatrace/dynatrace-operator/src/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/daemonset"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	DaemonSetName = "dynatrace-oneagent-csi-driver"
)

func Get(ctx context.Context, resource *resources.Resources, namespace string) (appsv1.DaemonSet, error) {
	return daemonset.NewQuery(ctx, resource, client.ObjectKey{
		Name:      DaemonSetName,
		Namespace: namespace,
	}).Get()
}

func ForEachPod(ctx context.Context, resource *resources.Resources, namespace string, consumer daemonset.PodConsumer) error {
	return daemonset.NewQuery(ctx, resource, client.ObjectKey{
		Name:      DaemonSetName,
		Namespace: namespace,
	}).ForEachPod(consumer)
}

func CleanUpEachPod(namespace string) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		resource := environmentConfig.Client().Resources()
		require.NoError(t, ForEachPod(ctx, resource, namespace, cleanUpPodConsumer(ctx, resource)))
		return ctx
	}
}

func WaitForDaemonset(namespace string) features.Func {
	return daemonset.WaitFor(DaemonSetName, namespace)
}

func cleanUpPodConsumer(ctx context.Context, resource *resources.Resources) daemonset.PodConsumer {
	return func(pod corev1.Pod) {
		resource.ExecInPod(ctx, pod.Namespace, pod.Name, "server", []string{"rm", "-rf", dtcsi.DataPath}, bytes.NewBuffer([]byte{}), bytes.NewBuffer([]byte{}))
	}
}

//go:build e2e

package csi

import (
	"context"
	"testing"

	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/daemonset"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/pod"
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

func CleanUpEachPod(namespace string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resource := envConfig.Client().Resources()
		require.NoError(t, daemonset.NewQuery(ctx, resource, client.ObjectKey{
			Name:      DaemonSetName,
			Namespace: namespace,
		}).ForEachPod(cleanUpPodConsumer(ctx, resource)))
		return ctx
	}
}

func WaitForDaemonset(namespace string) features.Func {
	return daemonset.WaitFor(DaemonSetName, namespace)
}

func cleanUpPodConsumer(ctx context.Context, resource *resources.Resources) daemonset.PodConsumer {
	return func(p corev1.Pod) {
		pod.Exec(ctx, resource, p, "server", "rm", "-rf", dtcsi.DataPath)
	}
}

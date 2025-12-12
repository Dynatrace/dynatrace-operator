//go:build e2e

package csi

import (
	"context"

	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubeobjects/daemonset"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubeobjects/pod"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

const (
	DaemonSetName = "dynatrace-oneagent-csi-driver"
)

func CleanUpEachPod(namespace string) env.Func {
	return func(ctx context.Context, envConfig *envconf.Config) (context.Context, error) {
		resource := envConfig.Client().Resources()

		return ctx, daemonset.NewQuery(ctx, resource, client.ObjectKey{
			Name:      DaemonSetName,
			Namespace: namespace,
		}).ForEachPod(cleanUpPodConsumer(ctx, resource))
	}
}

func WaitForDaemonset(namespace string) env.Func {
	return daemonset.WaitFor(DaemonSetName, namespace)
}

func cleanUpPodConsumer(ctx context.Context, resource *resources.Resources) daemonset.PodConsumer {
	return func(p corev1.Pod) {
		pod.Exec(ctx, resource, p, "server", "rm", "-rf", dtcsi.DataPath)
	}
}

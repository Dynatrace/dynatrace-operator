//go:build e2e

package oneagent

import (
	"context"
	"testing"
	"time"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/daemonset"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/pod"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func WaitForDaemonset(dynakube dynatracev1beta1.DynaKube) features.Func {
	return daemonset.WaitFor(dynakube.OneAgentDaemonsetName(), dynakube.Namespace)
}

func WaitForDaemonSetPodsDeletion(dynakube dynatracev1beta1.DynaKube) features.Func {
	return pod.WaitForPodsDeletionWithOwner(dynakube.OneAgentDaemonsetName(), dynakube.Namespace)
}

func Get(ctx context.Context, resource *resources.Resources, dynakube dynatracev1beta1.DynaKube) (appsv1.DaemonSet, error) {
	return daemonset.NewQuery(ctx, resource, client.ObjectKey{
		Name:      dynakube.OneAgentDaemonsetName(),
		Namespace: dynakube.Namespace,
	}).Get()
}

func ForEachPod(ctx context.Context, resource *resources.Resources, dynakube dynatracev1beta1.DynaKube, actionFunc daemonset.PodConsumer) error {
	return daemonset.NewQuery(ctx, resource, client.ObjectKey{
		Name:      dynakube.OneAgentDaemonsetName(),
		Namespace: dynakube.Namespace,
	}).ForEachPod(actionFunc)
}

func OneAgentPodsAreReady(dynakube dynatracev1beta1.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		resource := environmentConfig.Client().Resources()

		require.NoError(t, ForEachPod(ctx, resource, dynakube, func(pod corev1.Pod) {
			err := wait.For(conditions.New(resource).PodReady(&pod), wait.WithTimeout(time.Minute*10))
			require.NoError(t, err)
		}))

		return ctx
	}
}

package daemonset

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func WaitFor(name string, namespace string) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		resources := environmentConfig.Client().Resources()
		err := wait.For(conditions.New(resources).ResourceMatch(&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		}, func(object k8s.Object) bool {
			daemonset, isDaemonset := object.(*appsv1.DaemonSet)
			return isDaemonset && daemonset.Status.DesiredNumberScheduled == daemonset.Status.NumberReady
		}))

		require.NoError(t, err)
		return ctx
	}
}

func WaitForPodsDeletion(ownerName string, namespace string) func(ctx context.Context, environmentConfig *envconf.Config, t *testing.T) (context.Context, error) {
	return func(ctx context.Context, environmentConfig *envconf.Config, t *testing.T) (context.Context, error) {
		var pods corev1.PodList
		resources := environmentConfig.Client().Resources()
		err := resources.WithNamespace(namespace).List(ctx, &pods)

		if err != nil {
			if k8serrors.IsNotFound(err) {
				err = nil
			}

			return ctx, errors.WithStack(err)
		}

		var targetPods corev1.PodList
		for _, pod := range pods.Items {
			if len(pod.ObjectMeta.OwnerReferences) < 1 {
				continue
			}

			if pod.ObjectMeta.OwnerReferences[0].Name == ownerName {
				targetPods.Items = append(targetPods.Items, pod)
			}
		}

		err = wait.For(conditions.New(resources).ResourcesDeleted(&targetPods))
		return ctx, errors.WithStack(err)
	}
}

//go:build e2e

package k8sstatefulset

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func Get(ctx context.Context, resource *resources.Resources, name, namespace string) (appsv1.StatefulSet, error) {
	var stateFulSet appsv1.StatefulSet
	err := resource.Get(ctx, name, namespace, &stateFulSet)

	return stateFulSet, err
}

func IsReady(name, namespace string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()
		sts, err := Get(ctx, resources, name, namespace)
		require.NoError(t, err)
		assert.Equal(t, sts.Status.Replicas, sts.Status.ReadyReplicas)

		return ctx
	}
}

// WaitFor wait until StatefulSet status replicas and readyReplicas are equal.
// For cases when resources should already be in this state, e.g. after the initial DynaKube install,
// [IsReady] should be used instead.
func WaitFor(name string, namespace string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()
		err := wait.For(conditions.New(resources).ResourceMatch(&appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		}, func(object k8s.Object) bool {
			statefulSet, isStatefulSet := object.(*appsv1.StatefulSet)

			return isStatefulSet && statefulSet.Status.Replicas == statefulSet.Status.ReadyReplicas
		}), wait.WithTimeout(10*time.Minute))
		// Default of 5 minutes can be a bit too short for the ActiveGate to startup

		require.NoError(t, err)

		return ctx
	}
}

func WaitForDeletion(name string, namespace string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()
		err := wait.For(conditions.New(resources).ResourceDeleted(&appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		}), wait.WithTimeout(2*time.Minute))

		require.NoError(t, err)

		return ctx
	}
}

//go:build e2e

package statefulset

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

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

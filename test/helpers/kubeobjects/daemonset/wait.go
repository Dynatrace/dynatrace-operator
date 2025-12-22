//go:build e2e

package daemonset

import (
	"context"
	"time"

	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func WaitFor(name string, namespace string) env.Func {
	return func(ctx context.Context, envConfig *envconf.Config) (context.Context, error) {
		resources := envConfig.Client().Resources()
		isReady := conditions.New(resources).DaemonSetReady(&appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}})
		err := wait.For(func(ctx context.Context) (done bool, err error) {
			done, err = isReady(ctx)
			// DaemonSets may not be immediately available when WaitFor is called.
			err = client.IgnoreNotFound(err)

			return
		}, wait.WithTimeout(10*time.Minute))

		return ctx, err
	}
}

// WaitForDaemonset wait until DaemonSet status numberReady and desiredNumberScheduled are equal.
// For cases when resources should already be in this state, e.g. after the initial DynaKube install,
// [IsReady] should be used instead.
func WaitForDaemonset(name string, namespace string) features.Func {
	return helpers.ToFeatureFunc(WaitFor(name, namespace), true)
}

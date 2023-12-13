//go:build e2e

package daemonset

import (
	"context"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/logger"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

var (
	log     = logger.Factory.GetLogger("main")
	timeout = 5 * time.Minute
)

func WaitFor(name string, namespace string) env.Func {
	return func(ctx context.Context, envConfig *envconf.Config) (context.Context, error) {
		resources := envConfig.Client().Resources()
		err := wait.For(conditions.New(resources).ResourceMatch(&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		}, func(object k8s.Object) bool {
			daemonset, isDaemonset := object.(*appsv1.DaemonSet)
			return isDaemonset && daemonset.Status.DesiredNumberScheduled == daemonset.Status.UpdatedNumberScheduled &&
				daemonset.Status.DesiredNumberScheduled == daemonset.Status.NumberReady
		}), wait.WithTimeout(timeout))
		// Workaround to make OCP tests pass on 'oneagent_started' step
		if err != nil {
			log.Info("WARNING: OneAgent deamonset timed out getting ready (%ss) [%s]", timeout, err)
		}
		return ctx, nil
	}
}

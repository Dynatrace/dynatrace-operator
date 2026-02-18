//go:build e2e

package k8sdaemonset

import (
	"context"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

type PodConsumer func(pod corev1.Pod)

type Query struct {
	ctx       context.Context
	resource  *resources.Resources
	objectKey client.ObjectKey
}

func NewQuery(ctx context.Context, resource *resources.Resources, objectKey client.ObjectKey) *Query {
	return &Query{
		ctx:       ctx,
		resource:  resource,
		objectKey: objectKey,
	}
}

func (query *Query) Get() (appsv1.DaemonSet, error) {
	var daemonSet appsv1.DaemonSet
	err := query.resource.Get(query.ctx, query.objectKey.Name, query.objectKey.Namespace, &daemonSet)

	return daemonSet, err
}

func (query *Query) ForEachPod(actionFunc PodConsumer) error {
	var pods corev1.PodList
	daemonSet, err := query.Get()

	if err != nil {
		return err
	}

	err = query.resource.List(query.ctx, &pods, resources.WithLabelSelector(labels.FormatLabels(daemonSet.Spec.Selector.MatchLabels)))

	if err != nil {
		return err
	}

	for _, pod := range pods.Items {
		actionFunc(pod)
	}

	return nil
}

func Delete(name, namespace string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()
		ds := &appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}}
		require.NoError(t, client.IgnoreNotFound(resources.Delete(ctx, ds)))

		return ctx
	}
}

func IsReady(name, namespace string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()
		ds := &appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}}
		ready, err := conditions.New(resources).DaemonSetReady(ds)(ctx)
		require.NoError(t, err)
		assert.True(t, ready)

		return ctx
	}
}

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

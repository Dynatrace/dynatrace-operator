//go:build e2e

package k8sdeployment

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
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

func (query *Query) ForEachPod(consumer PodConsumer) error {
	deployment := &appsv1.Deployment{}
	if err := query.resource.Get(query.ctx, query.objectKey.Name, query.objectKey.Namespace, deployment); err != nil {
		return err
	}

	var pods corev1.PodList
	if err := query.resource.List(query.ctx, &pods, resources.WithLabelSelector(labels.FormatLabels(deployment.Spec.Selector.MatchLabels))); err != nil {
		return err
	}

	for _, pod := range pods.Items {
		consumer(pod)
	}

	return nil
}

func IsReady(name, namespace string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()
		deploy := &appsv1.Deployment{}
		require.NoError(t, resources.Get(ctx, name, namespace, deploy))
		assert.Equal(t, deploy.Status.Replicas, deploy.Status.ReadyReplicas)

		return ctx
	}
}

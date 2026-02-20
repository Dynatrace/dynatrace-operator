//go:build e2e

package k8sdeployment

import (
	"context"
	"testing"
	"time"

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

const DeploymentAvailableTimeout = 5 * time.Minute

const DeploymentReplicaFailureTimeout = 5 * time.Minute

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
		deployment := &appsv1.Deployment{}
		require.NoError(t, resources.Get(ctx, name, namespace, deployment))
		assert.Equal(t, deployment.Status.Replicas, deployment.Status.ReadyReplicas)

		return ctx
	}
}

func WaitFor(name string, namespace string) env.Func {
	return func(ctx context.Context, envConfig *envconf.Config) (context.Context, error) {
		clientResources := envConfig.Client().Resources()
		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		}

		return ctx, WaitUntilReady(clientResources, deployment)
	}
}

func WaitUntilReady(resource *resources.Resources, deployment *appsv1.Deployment) error {
	return wait.For(conditions.New(resource).DeploymentConditionMatch(deployment, appsv1.DeploymentAvailable, corev1.ConditionTrue), wait.WithTimeout(DeploymentAvailableTimeout))
}

func WaitUntilFailedCreate(resource *resources.Resources, deployment *appsv1.Deployment) error {
	return wait.For(conditions.New(resource).DeploymentConditionMatch(deployment, appsv1.DeploymentReplicaFailure, corev1.ConditionTrue), wait.WithTimeout(DeploymentReplicaFailureTimeout))
}

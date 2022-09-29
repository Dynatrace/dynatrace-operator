package sampleapps

import (
	"context"
	"github.com/stretchr/testify/require"
	"k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"testing"
)

const (
	Name      = "myapp"
	Namespace = "test-namespace-1"
)

func Restart(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
	var sampleDeployment v1.Deployment
	var pods corev1.PodList
	resource := config.Client().Resources()

	require.NoError(t, resource.WithNamespace(Namespace).List(ctx, &pods))

	for _, podItem := range pods.Items {
		require.NoError(t, resource.Delete(ctx, &podItem))
	}

	require.NoError(t, resource.Get(ctx, Name, Namespace, &sampleDeployment))
	require.NoError(t, wait.For(
		conditions.New(resource).DeploymentConditionMatch(
			&sampleDeployment, v1.DeploymentAvailable, corev1.ConditionTrue)))

	return ctx
}

func Get(t *testing.T, ctx context.Context, resource *resources.Resources) corev1.PodList {
	var pods corev1.PodList

	require.NoError(t, resource.WithNamespace(Namespace).List(ctx, &pods))
	return pods
}

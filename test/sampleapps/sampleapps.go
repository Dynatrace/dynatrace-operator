package sampleapps

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

const (
	Name           = "myapp"
	Namespace      = "test-namespace-1"
	AdaptedTimeOut = time.Minute * 10
)

func Install(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
	var sampleDeployment appsv1.Deployment
	resource := config.Client().Resources()

	require.NoError(t, resource.Get(ctx, Name, Namespace, &sampleDeployment))
	require.NoError(t, wait.For(
		conditions.New(resource).DeploymentConditionMatch(
			&sampleDeployment, appsv1.DeploymentAvailable, corev1.ConditionTrue), wait.WithTimeout(AdaptedTimeOut)))

	return ctx
}

func RestartHalf(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
	deleteFn := func(t *testing.T, ctx context.Context, pods corev1.PodList, resource *resources.Resources) {
		for i, podItem := range pods.Items {
			if i%2 == 1 {
				continue // skip odd-indexed pods
			}
			require.NoError(t, resource.Delete(ctx, &podItem)) //nolint:gosec
		}
	}

	return doRestart(ctx, t, config, deleteFn)
}

func Restart(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
	deleteFn := func(t *testing.T, ctx context.Context, pods corev1.PodList, resource *resources.Resources) {
		for _, podItem := range pods.Items {
			require.NoError(t, resource.Delete(ctx, &podItem)) //nolint:gosec
		}
	}

	return doRestart(ctx, t, config, deleteFn)
}

func doRestart(ctx context.Context, t *testing.T, config *envconf.Config, deleteFn func(t *testing.T, ctx context.Context, pods corev1.PodList, resource *resources.Resources)) context.Context {
	var sampleDeployment appsv1.Deployment
	var pods corev1.PodList
	resource := config.Client().Resources()

	require.NoError(t, resource.WithNamespace(Namespace).List(ctx, &pods))

	deleteFn(t, ctx, pods, resource)

	require.NoError(t, resource.Get(ctx, Name, Namespace, &sampleDeployment))
	require.NoError(t, wait.For(
		conditions.New(resource).DeploymentConditionMatch(
			&sampleDeployment, appsv1.DeploymentAvailable, corev1.ConditionTrue)))

	return ctx
}

func Get(ctx context.Context, t *testing.T, resource *resources.Resources) corev1.PodList {
	var pods corev1.PodList

	require.NoError(t, resource.WithNamespace(Namespace).List(ctx, &pods))
	return pods
}

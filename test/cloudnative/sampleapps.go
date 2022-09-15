package cloudnative

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

const (
	sampleAppsName      = "myapp"
	sampleAppsNamespace = "test-namespace-1"
)

func restartSampleApps(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
	var sampleDeployment appsv1.Deployment
	var pods corev1.PodList
	resources := config.Client().Resources()

	require.NoError(t, resources.WithNamespace(sampleAppsNamespace).List(ctx, &pods))

	for _, podItem := range pods.Items {
		require.NoError(t, resources.Delete(ctx, &podItem))
	}

	require.NoError(t, resources.Get(ctx, sampleAppsName, sampleAppsNamespace, &sampleDeployment))
	require.NoError(t, wait.For(
		conditions.New(resources).DeploymentConditionMatch(
			&sampleDeployment, appsv1.DeploymentAvailable, corev1.ConditionTrue)))

	return ctx
}

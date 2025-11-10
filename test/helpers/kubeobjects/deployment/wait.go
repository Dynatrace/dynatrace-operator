//go:build e2e

package deployment

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const DeploymentAvailableTimeout = 5 * time.Minute

const DeploymentReplicaFailureTimeout = 5 * time.Minute

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

func WaitForReplicas(name string, namespace string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()
		err := wait.For(conditions.New(resources).ResourceMatch(&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		}, func(object k8s.Object) bool {
			deployment, isDeployment := object.(*appsv1.Deployment)

			return isDeployment && deployment.Status.Replicas == deployment.Status.ReadyReplicas
		}), wait.WithTimeout(10*time.Minute))

		require.NoError(t, err)

		return ctx
	}
}

func WaitUntilReady(resource *resources.Resources, deployment *appsv1.Deployment) error {
	return wait.For(conditions.New(resource).DeploymentConditionMatch(deployment, appsv1.DeploymentAvailable, corev1.ConditionTrue), wait.WithTimeout(DeploymentAvailableTimeout))
}

func WaitUntilFailedCreate(resource *resources.Resources, deployment *appsv1.Deployment) error {
	return wait.For(conditions.New(resource).DeploymentConditionMatch(deployment, appsv1.DeploymentReplicaFailure, corev1.ConditionTrue), wait.WithTimeout(DeploymentReplicaFailureTimeout))
}

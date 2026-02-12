//go:build e2e

package k8sdeployment

import (
	"context"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
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

func WaitUntilReady(resource *resources.Resources, deployment *appsv1.Deployment) error {
	return wait.For(conditions.New(resource).DeploymentConditionMatch(deployment, appsv1.DeploymentAvailable, corev1.ConditionTrue), wait.WithTimeout(DeploymentAvailableTimeout))
}

func WaitUntilFailedCreate(resource *resources.Resources, deployment *appsv1.Deployment) error {
	return wait.For(conditions.New(resource).DeploymentConditionMatch(deployment, appsv1.DeploymentReplicaFailure, corev1.ConditionTrue), wait.WithTimeout(DeploymentReplicaFailureTimeout))
}

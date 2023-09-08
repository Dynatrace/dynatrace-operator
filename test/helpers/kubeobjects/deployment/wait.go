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
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const DeploymentAvailableTimeout = 15 * time.Minute

func WaitFor(name string, namespace string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()
		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		}
		err := WaitUntilReady(resources, deployment)

		require.NoError(t, err)
		return ctx
	}
}

func WaitUntilReady(resource *resources.Resources, deployment *appsv1.Deployment) error {
	return wait.For(conditions.New(resource).DeploymentConditionMatch(deployment, appsv1.DeploymentAvailable, corev1.ConditionTrue), wait.WithTimeout(DeploymentAvailableTimeout))
}

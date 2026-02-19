//go:build e2e

package k8spod

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func List(ctx context.Context, t *testing.T, resource *resources.Resources, namespace string) corev1.PodList {
	var pods corev1.PodList

	require.NoError(t, resource.WithNamespace(namespace).List(ctx, &pods))

	return pods
}

func ListForOwner(ctx context.Context, t *testing.T, resource *resources.Resources, ownerName, namespace string) corev1.PodList {
	pods := List(ctx, t, resource, namespace)

	var targetPods corev1.PodList
	for _, pod := range pods.Items {
		if len(pod.OwnerReferences) < 1 {
			continue
		}

		if pod.OwnerReferences[0].Name == ownerName {
			targetPods.Items = append(targetPods.Items, pod)
		}
	}

	return targetPods
}

type ConditionFunction func(object k8s.Object) bool

func WaitForCondition(name string, namespace string, conditionFunction ConditionFunction, timeout time.Duration) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()

		err := wait.For(conditions.New(resources).ResourceMatch(&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		}, conditionFunction), wait.WithTimeout(timeout))

		require.NoError(t, err)

		return ctx
	}
}

func WaitFor(name string, namespace string) features.Func {
	return WaitForCondition(name, namespace, func(object k8s.Object) bool {
		pod, isPod := object.(*corev1.Pod)

		return isPod && (pod.Status.Phase == corev1.PodRunning || pod.Status.Phase == corev1.PodSucceeded)
	}, 10*time.Minute)
	// Default of 5 minutes can be a bit too short for the ActiveGate to startup
}

func WaitForDeletionWithOwner(ownerName string, namespace string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()
		targetPods := ListForOwner(ctx, t, resources, ownerName, namespace)

		err := wait.For(conditions.New(resources).ResourcesDeleted(&targetPods), wait.WithTimeout(5*time.Minute))
		require.NoError(t, err)

		return ctx
	}
}

type ExecutionResult struct {
	StdOut *bytes.Buffer
	StdErr *bytes.Buffer
}

func Exec(ctx context.Context, resource *resources.Resources, pod corev1.Pod, container string, command ...string) (*ExecutionResult, error) {
	result := &ExecutionResult{
		StdOut: &bytes.Buffer{},
		StdErr: &bytes.Buffer{},
	}

	err := resource.ExecInPod(
		ctx,
		pod.Namespace,
		pod.Name,
		container,
		command,
		result.StdOut,
		result.StdErr,
	)

	if err != nil {
		return result, errors.WithMessagef(errors.WithStack(err),
			"stdout:\n%s\nstderr:\n%s", result.StdOut.String(), result.StdErr.String())
	}

	return result, nil
}

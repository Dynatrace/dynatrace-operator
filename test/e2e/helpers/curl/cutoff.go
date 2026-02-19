//go:build e2e

package curl

import (
	"context"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8spod"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func InstallCutOffCurlPod(podName, namespaceName, curlTarget string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		require.NoError(t, envConfig.Client().Resources().Create(ctx, buildCutOffCurlPod(podName, namespaceName, curlTarget)))

		return ctx
	}
}

func WaitForCutOffCurlPod(podName, namespaceName string) features.Func {
	return k8spod.WaitForCondition(podName, namespaceName, func(object k8s.Object) bool {
		pod, isPod := object.(*corev1.Pod)
		// If probe fails we don't have internet, so we achieve waiting condition
		return isPod && !pod.Status.ContainerStatuses[0].Ready
	}, connectionTimeout*2*time.Second)
}

func DeleteCutOffCurlPod(podName, namespaceName, curlTarget string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		err := envConfig.Client().Resources().Delete(ctx, buildCutOffCurlPod(podName, namespaceName, curlTarget))
		if !k8sErrors.IsNotFound(err) {
			require.NoError(t, err)
		}

		return ctx
	}
}

func buildCutOffCurlPod(podName, namespaceName, curlTarget string) *corev1.Pod {
	probe := corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			Exec: &corev1.ExecAction{
				Command: []string{"curl", curlTarget, "--insecure", "--verbose", "--head", "--fail"},
			},
		},
		InitialDelaySeconds: 30,
		PeriodSeconds:       30,
		FailureThreshold:    2,
		SuccessThreshold:    3,
		TimeoutSeconds:      3,
	}
	options := []Option{
		WithCommand([]string{"sleep"}),
		WithArgs([]string{"120"}),
		WithReadinessProbe(&probe),
		WithRestartPolicy(corev1.RestartPolicyNever),
	}

	return NewPod(podName, namespaceName, curlTarget, options...)
}

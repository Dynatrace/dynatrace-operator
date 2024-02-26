//go:build e2e

package curl

import (
	"context"
	"io"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
)

const (
	curlContainerName = "curl"
	connectionTimeout = 5
)

func GetCurlPodLogStream(ctx context.Context, t *testing.T, resources *resources.Resources,
	podName string, namespace string) io.ReadCloser {
	clientset, err := kubernetes.NewForConfig(resources.GetConfig())
	require.NoError(t, err)

	logStream, err := clientset.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{
		Container: curlContainerName,
	}).Stream(ctx)
	require.NoError(t, err)

	return logStream
}

type Option func(curlPod *corev1.Pod)

func NewPod(podName, namespaceName, targetUrl string, options ...Option) *corev1.Pod {
	curlPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespaceName,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  curlContainerName,
					Image: "curlimages/curl",
					Command: []string{
						"curl",
					},
					Args: []string{
						targetUrl,
						"--insecure",
						"--verbose",
					},
				},
			},
			RestartPolicy: corev1.RestartPolicyOnFailure,
		},
	}
	for _, opt := range options {
		opt(curlPod)
	}

	return curlPod
}

func WithCommand(command []string) Option {
	return func(curlPod *corev1.Pod) {
		curlPod.Spec.Containers[0].Command = command
	}
}

func WithArgs(args []string) Option {
	return func(curlPod *corev1.Pod) {
		curlPod.Spec.Containers[0].Args = args
	}
}

func WithReadinessProbe(probe *corev1.Probe) Option {
	return func(curlPod *corev1.Pod) {
		curlPod.Spec.Containers[0].ReadinessProbe = probe
	}
}

func WithProxy(dynakube dynatracev1beta1.DynaKube) Option {
	return func(curlPod *corev1.Pod) {
		if dynakube.HasProxy() {
			proxyEnv := corev1.EnvVar{
				Name:  "https_proxy",
				Value: dynakube.Spec.Proxy.Value,
			}
			curlPod.Spec.Containers[0].Env = append(curlPod.Spec.Containers[0].Env, proxyEnv)
		}
	}
}

func WithRestartPolicy(restartPolicy corev1.RestartPolicy) Option {
	return func(curlPod *corev1.Pod) {
		curlPod.Spec.RestartPolicy = restartPolicy
	}
}

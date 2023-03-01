//go:build e2e

package sampleapps

import (
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/webhook"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/logs"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	activeGateEndpoint = "rest/state"
	livezEndpoint      = "livez"

	curlPodNameActivegate = "curl-activegate"
	curlPodNameWebhook    = "curl-webhook"
	curlContainerName     = "curl"
)

func InstallActiveGateCurlPod(dynakube dynatracev1beta1.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		serviceUrl := getActiveGateServiceUrl(dynakube)
		curlTarget := fmt.Sprintf("%s/%s", serviceUrl, activeGateEndpoint)

		curlPod := setupCurlPod(dynakube, curlPodNameActivegate, curlTarget)
		require.NoError(t, environmentConfig.Client().Resources().Create(ctx, curlPod))
		return ctx
	}
}

func WaitForActiveGateCurlPod(dynakube dynatracev1beta1.DynaKube) features.Func {
	return pod.WaitFor(curlPodNameActivegate, dynakube.Namespace)
}

func CheckActiveGateCurlResult(dynakube dynatracev1beta1.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		resources := environmentConfig.Client().Resources()

		logStream := getCurlPodLogStream(ctx, t, resources, curlPodNameActivegate, dynakube.Namespace)
		logs.AssertContains(t, logStream, "RUNNING")

		return ctx
	}
}

func InstallWebhookCurlProxyPod(dynakube dynatracev1beta1.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		curlTarget := fmt.Sprintf("https://%s/%s", getWebhookServiceUrl(dynakube), livezEndpoint)
		curlPod := setupCurlPod(dynakube, curlPodNameWebhook, curlTarget)
		require.NoError(t, environmentConfig.Client().Resources().Create(ctx, curlPod))

		return ctx
	}
}

func WaitForWebhookCurlProxyPod(dynakube dynatracev1beta1.DynaKube) features.Func {
	return pod.WaitForCondition(curlPodNameWebhook, dynakube.Namespace, func(object k8s.Object) bool {
		pod, isPod := object.(*corev1.Pod)
		return isPod && pod.Status.Phase == corev1.PodSucceeded
	}, 30*time.Second)
}

func CheckWebhookCurlProxyResult(dynakube dynatracev1beta1.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		resources := environmentConfig.Client().Resources()

		logStream := getCurlPodLogStream(ctx, t, resources, curlPodNameWebhook, dynakube.Namespace)

		webhookServiceUrl := getWebhookServiceUrl(dynakube)
		logs.AssertContains(t, logStream, fmt.Sprintf("CONNECT %s:443", webhookServiceUrl))

		return ctx
	}
}

func getActiveGateServiceUrl(dynakube dynatracev1beta1.DynaKube) string {
	serviceName := capability.BuildServiceName(dynakube.Name, consts.MultiActiveGateName)
	return fmt.Sprintf("https://%s.%s.svc.cluster.local", serviceName, dynakube.Namespace)
}

func getWebhookServiceUrl(dynakube dynatracev1beta1.DynaKube) string {
	return fmt.Sprintf("%s.%s.svc.cluster.local", webhook.DeploymentName, dynakube.Namespace)
}

func getCurlPodLogStream(ctx context.Context, t *testing.T, resources *resources.Resources,
	podName string, namespace string) io.ReadCloser {
	clientset, err := kubernetes.NewForConfig(resources.GetConfig())
	require.NoError(t, err)

	logStream, err := clientset.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{
		Container: curlContainerName,
	}).Stream(ctx)
	require.NoError(t, err)

	return logStream
}

func setupCurlPod(dynakube dynatracev1beta1.DynaKube, podName, targetUrl string) *corev1.Pod {
	curlPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: dynakube.Namespace,
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

	if dynakube.HasProxy() {
		proxyEnv := corev1.EnvVar{
			Name:  "https_proxy",
			Value: dynakube.Spec.Proxy.Value,
		}
		curlPod.Spec.Containers[0].Env = append(curlPod.Spec.Containers[0].Env, proxyEnv)
	}
	return curlPod
}

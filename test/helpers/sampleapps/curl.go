//go:build e2e

package sampleapps

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"testing"
	"time"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
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

	CurlPodNameActivegateHttps = "curl-activegate-https"
	CurlPodNameActivegateHttp  = "curl-activegate-http"
	curlContainerName          = "curl"

	connectionTimeout = 5

	proxyNamespaceName = "proxy"
)

func InstallActiveGateCurlPod(dynakube dynatracev1beta1.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		serviceUrl := getActiveGateServiceUrl(dynakube)
		curlTarget := fmt.Sprintf("%s/%s", serviceUrl, activeGateEndpoint)

		curlPod := NewCurlPodBuilder(CurlPodNameActivegateHttps, curlNamespace(dynakube), curlTarget).WithProxy(dynakube).Build()
		require.NoError(t, environmentConfig.Client().Resources().Create(ctx, curlPod))
		return ctx
	}
}

func InstallActiveGateHttpCurlPod(dynakube dynatracev1beta1.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		serviceUrl := getActiveGateHttpServiceUrl(dynakube)
		curlTarget := fmt.Sprintf("%s/%s", serviceUrl, activeGateEndpoint)

		curlPod := NewCurlPodBuilder(CurlPodNameActivegateHttp, curlNamespace(dynakube), curlTarget).WithProxy(dynakube).Build()
		require.NoError(t, environmentConfig.Client().Resources().Create(ctx, curlPod))
		return ctx
	}
}

func WaitForActiveGateCurlPod(podName string, dynakube dynatracev1beta1.DynaKube) features.Func {
	return pod.WaitFor(podName, curlNamespace(dynakube))
}

func CheckActiveGateCurlResult(podName string, dynakube dynatracev1beta1.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		resources := environmentConfig.Client().Resources()

		logStream := getCurlPodLogStream(ctx, t, resources, podName, curlNamespace(dynakube))
		logs.AssertContains(t, logStream, "RUNNING")

		return ctx
	}
}

func curlNamespace(dynakube dynatracev1beta1.DynaKube) string {
	if dynakube.HasProxy() {
		return proxyNamespaceName
	}
	return dynakube.Namespace
}

func getActiveGateServiceUrl(dynakube dynatracev1beta1.DynaKube) string {
	serviceName := capability.BuildServiceName(dynakube.Name, consts.MultiActiveGateName)
	return fmt.Sprintf("https://%s.%s.svc.cluster.local", serviceName, dynakube.Namespace)
}

func getActiveGateHttpServiceUrl(dynakube dynatracev1beta1.DynaKube) string {
	serviceName := capability.BuildServiceName(dynakube.Name, consts.MultiActiveGateName)
	return fmt.Sprintf("http://%s.%s.svc.cluster.local", serviceName, dynakube.Namespace)
}

func GetWebhookServiceUrl(dynakube dynatracev1beta1.DynaKube) string {
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

func InstallCutOffCurlPod(podName, namespaceName, curlTarget string) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		// if curl command can't connect to the host, returns 28 after 131[s] by default
		curlPod := NewCurlPodBuilder(podName, namespaceName, curlTarget).WithRestartPolicy(corev1.RestartPolicyNever).WithParameters("--connect-timeout", strconv.Itoa(connectionTimeout)).Build()
		require.NoError(t, environmentConfig.Client().Resources().Create(ctx, curlPod))
		return ctx
	}
}

func WaitForCutOffCurlPod(podName, namespaceName string) features.Func {
	return pod.WaitForCondition(podName, namespaceName, func(object k8s.Object) bool {
		pod, isPod := object.(*corev1.Pod)
		// kubernetes 28
		// openshift 7
		return isPod && pod.Status.ContainerStatuses[0].State.Terminated != nil && (pod.Status.ContainerStatuses[0].State.Terminated.ExitCode == 28 || pod.Status.ContainerStatuses[0].State.Terminated.ExitCode == 7)
	}, connectionTimeout*2*time.Second)
}

type CurlPodBuilder struct {
	curlPod *corev1.Pod
}

func NewCurlPodBuilder(podName, namespaceName, targetUrl string) CurlPodBuilder {
	return CurlPodBuilder{
		curlPod: &corev1.Pod{
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
		},
	}
}

func (curlPodBuilder CurlPodBuilder) WithProxy(dynakube dynatracev1beta1.DynaKube) CurlPodBuilder {
	if dynakube.HasProxy() {
		proxyEnv := corev1.EnvVar{
			Name:  "https_proxy",
			Value: dynakube.Spec.Proxy.Value,
		}
		curlPodBuilder.curlPod.Spec.Containers[0].Env = append(curlPodBuilder.curlPod.Spec.Containers[0].Env, proxyEnv)
	}
	return curlPodBuilder
}

func (curlPodBuilder CurlPodBuilder) WithRestartPolicy(restartPolicy corev1.RestartPolicy) CurlPodBuilder {
	curlPodBuilder.curlPod.Spec.RestartPolicy = restartPolicy
	return curlPodBuilder
}

func (curlPodBuilder CurlPodBuilder) WithParameters(params ...string) CurlPodBuilder {
	curlPodBuilder.curlPod.Spec.Containers[0].Args = append(curlPodBuilder.curlPod.Spec.Containers[0].Args, params...)
	return curlPodBuilder
}

func (curlPodBuilder CurlPodBuilder) Build() *corev1.Pod {
	return curlPodBuilder.curlPod
}

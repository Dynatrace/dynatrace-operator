//go:build e2e

package sampleapps

import (
	"context"
	"fmt"
	"io"
	"path"
	"testing"
	"time"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/webhook"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/manifests"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/logs"
	"github.com/Dynatrace/dynatrace-operator/test/project"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	activeGateEndpoint = "rest/state"
	livezEndpoint      = "livez"
)

var (
	curlPodTemplatePath = path.Join(project.TestDataDir(), "network/curl-pod.yaml")
)

func InstallActiveGateCurlPod(testDynakube dynatracev1beta1.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		pod := manifests.ObjectFromFile[*corev1.Pod](t, curlPodTemplatePath)

		serviceUrl := getActiveGateServiceUrl(testDynakube)
		curlEndpoint := fmt.Sprintf("%s/%s", serviceUrl, activeGateEndpoint)

		podName := getCurlPodName(testDynakube)
		pod.Name = podName
		pod.Spec.Containers[0].Name = podName

		if testDynakube.HasProxy() {
			proxyArguments := []string{}
			proxyArguments = append(proxyArguments, testDynakube.Spec.Proxy.Value)
			proxyArguments = append(proxyArguments, "-o", "/dev/null")
			proxyArguments = append(proxyArguments, "-v", "--max-time", "4")
			pod.Spec.Containers[0].Args = append(pod.Spec.Containers[0].Args, proxyArguments...)

			probeEndpoint := fmt.Sprintf("%s%s", serviceUrl, livezEndpoint)
			curlEndpoint = probeEndpoint

			probeCommand := []string{"curl", probeEndpoint, "-k"}
			probeCommand = append(probeCommand, proxyArguments...)

			pod.Spec.Containers[0].LivenessProbe = &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					Exec: &corev1.ExecAction{
						Command: probeCommand,
					},
				},
				InitialDelaySeconds: 5,
				PeriodSeconds:       5,
				FailureThreshold:    1,
			}
			pod.Spec.RestartPolicy = corev1.RestartPolicyOnFailure
		}

		pod.Namespace = testDynakube.Namespace
		pod.Spec.Containers[0].Args[0] = curlEndpoint

		require.NoError(t, environmentConfig.Client().Resources().Create(ctx, pod))
		return ctx
	}
}

func WaitForActiveGateCurlPod(testDynakube dynatracev1beta1.DynaKube) features.Func {
	return pod.WaitFor(getCurlPodName(testDynakube), testDynakube.Namespace)
}

func CheckActiveGateCurlResult(testDynakube dynatracev1beta1.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		resources := environmentConfig.Client().Resources()

		logStream := getCurlPodLogStream(ctx, t, resources, testDynakube)

		logs.AssertContains(t, logStream, "RUNNING")

		return ctx
	}
}

func InstallWebhookCurlProxyPod(testDynakube dynatracev1beta1.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		pod := manifests.ObjectFromFile[*corev1.Pod](t, curlPodTemplatePath)

		serviceUrl := getWebhookServiceUrl(testDynakube)
		curlEndpoint := fmt.Sprintf("%s/%s", serviceUrl, livezEndpoint)

		podName := getCurlPodName(testDynakube)
		pod.Name = podName
		pod.Spec.Containers[0].Name = podName

		if testDynakube.HasProxy() {
			proxyArguments := []string{}
			proxyArguments = append(proxyArguments, testDynakube.Spec.Proxy.Value)
			proxyArguments = append(proxyArguments, "-o", "/dev/null")
			proxyArguments = append(proxyArguments, "-v", "--max-time", "4")
			pod.Spec.Containers[0].Args = append(pod.Spec.Containers[0].Args, proxyArguments...)

			probeEndpoint := curlEndpoint
			probeCommand := []string{"curl", probeEndpoint, "-k"}
			probeCommand = append(probeCommand, proxyArguments...)

			pod.Spec.Containers[0].LivenessProbe = &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					Exec: &corev1.ExecAction{
						Command: probeCommand,
					},
				},
				InitialDelaySeconds: 5,
				PeriodSeconds:       5,
				FailureThreshold:    1,
			}
			pod.Spec.RestartPolicy = corev1.RestartPolicyOnFailure
		}

		pod.Namespace = testDynakube.Namespace
		pod.Spec.Containers[0].Args[0] = curlEndpoint

		require.NoError(t, environmentConfig.Client().Resources().Create(ctx, pod))
		return ctx
	}
}

func WaitForWebhookCurlProxyPod(testDynakube dynatracev1beta1.DynaKube) features.Func {
	return pod.WaitForCondition(getCurlPodName(testDynakube), testDynakube.Namespace, func(object k8s.Object) bool {
		pod, isPod := object.(*corev1.Pod)
		return isPod && pod.Status.Phase == corev1.PodSucceeded
	}, 30*time.Second)
}

func CheckWebhookCurlProxyResult(testDynakube dynatracev1beta1.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		resources := environmentConfig.Client().Resources()

		logStream := getCurlPodLogStream(ctx, t, resources, testDynakube)

		webhookServiceUrl := getWebhookServiceUrl(testDynakube)
		logs.AssertContains(t, logStream, fmt.Sprintf("CONNECT %s", webhookServiceUrl))

		return ctx
	}
}

func getCurlPodName(testDynakube dynatracev1beta1.DynaKube) string {
	return fmt.Sprintf("curl-%s", testDynakube.Name)
}

func getActiveGateServiceUrl(testDynakube dynatracev1beta1.DynaKube) string {
	serviceName := capability.BuildServiceName(testDynakube.Name, consts.MultiActiveGateName)
	return fmt.Sprintf("https://%s.%s.svc.cluster.local", serviceName, testDynakube.Namespace)
}

func getWebhookServiceUrl(testDynakube dynatracev1beta1.DynaKube) string {
	return fmt.Sprintf("https://%s.%s.svc.cluster.local", webhook.DeploymentName, testDynakube.Namespace)
}

func getCurlPodLogStream(ctx context.Context, t *testing.T, resources *resources.Resources, testDynakube dynatracev1beta1.DynaKube) io.ReadCloser {
	clientset, err := kubernetes.NewForConfig(resources.GetConfig())
	require.NoError(t, err)

	curlPodName := getCurlPodName(testDynakube)
	logStream, err := clientset.CoreV1().Pods(testDynakube.Namespace).GetLogs(curlPodName, &corev1.PodLogOptions{
		Container: curlPodName,
	}).Stream(ctx)
	require.NoError(t, err)

	return logStream
}

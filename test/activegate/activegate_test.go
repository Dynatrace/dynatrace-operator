//go:build e2e

package activegate

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/test/csi"
	"github.com/Dynatrace/dynatrace-operator/test/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/deployment"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/environment"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/manifests"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/logs"
	"github.com/Dynatrace/dynatrace-operator/test/oneagent"
	"github.com/Dynatrace/dynatrace-operator/test/operator"
	"github.com/Dynatrace/dynatrace-operator/test/secrets"
	"github.com/Dynatrace/dynatrace-operator/test/webhook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	proxyNamespace  = "proxy"
	proxyDeployment = "squid"

	agNamespace = "dynatrace"
	agPodName   = "dynakube-activegate-0"

	agContainerName       = "activegate"
	agEecContainerName    = "activegate-eec"
	agStatsdContainerName = "activegate-statsd"

	curlPod = "curl"
)

var (
	agContainers = map[string]bool{
		agContainerName:       false,
		agStatsdContainerName: false,
		agEecContainerName:    false,
	}

	agInitContainers = map[string]bool{
		"certificate-loader": false,
	}

	agMounts = map[string][]string{
		agContainerName: {
			"/dev/root /var/lib/dynatrace/gateway/config",
			"tmpfs /var/lib/dynatrace/secrets/tokens/tenant-token",
			"tmpfs /var/lib/dynatrace/secrets/tokens/auth-token",
			"/dev/root /var/lib/dynatrace/remotepluginmodule/log/extensions/eec",
			"/dev/root /var/lib/dynatrace/remotepluginmodule/log/extensions/statsd",
			"/dev/root /opt/dynatrace/gateway/jre/lib/security/cacerts",
		},

		agEecContainerName: {
			"/dev/root /var/lib/dynatrace/gateway/config",
			"/dev/root /var/lib/dynatrace/remotepluginmodule/log/extensions",
			"/dev/root /opt/dynatrace/remotepluginmodule/agent/datasources/statsd",
			"/dev/root /var/lib/dynatrace/remotepluginmodule/log/extensions/datasources-statsd",
			"/dev/root /var/lib/dynatrace/remotepluginmodule/agent/conf/runtime",
			"/dev/root /var/lib/dynatrace/remotepluginmodule/agent/runtime/datasources",
		},
		agStatsdContainerName: {
			"/dev/root /var/lib/dynatrace/remotepluginmodule/agent/runtime/datasources",
			"/dev/root /var/lib/dynatrace/remotepluginmodule/log/extensions/datasources-statsd",
		},
	}
)

var testEnvironment env.Environment

func TestMain(m *testing.M) {
	testEnvironment = environment.Get()
	testEnvironment.BeforeEachTest(dynakube.DeleteDynakubeIfExists())
	testEnvironment.BeforeEachTest(oneagent.WaitForDaemonSetPodsDeletion())
	testEnvironment.BeforeEachTest(namespace.Recreate(dynakube.DynatraceNamespace))
	testEnvironment.AfterEachTest(namespace.DeleteIfExists(proxyNamespace))

	testEnvironment.AfterEachTest(dynakube.DeleteDynakubeIfExists())
	testEnvironment.AfterEachTest(oneagent.WaitForDaemonSetPodsDeletion())
	testEnvironment.AfterEachTest(namespace.Delete(dynakube.DynatraceNamespace))
	testEnvironment.AfterEachTest(namespace.DeleteIfExists(proxyNamespace))

	testEnvironment.Run(m)
}

func TestActiveGate(t *testing.T) {
	feature := install(t, nil)
	assessActiveGate(feature)
	testEnvironment.Test(t, feature.Feature())
}

func TestActiveGateProxy(t *testing.T) {
	feature := install(t, &v1beta1.DynaKubeProxy{
		Value: "http://squid.proxy:3128",
	})
	installProxy(feature)
	assessActiveGate(feature)
	testEnvironment.Test(t, feature.Feature())
}

func install(t *testing.T, proxy *v1beta1.DynaKubeProxy) *features.FeatureBuilder {
	secretConfig := dynakube.GetSecretConfig(t)

	defaultInstallation := features.New("capabilities")

	installAndDeploy(defaultInstallation, secretConfig)
	assessDeployment(defaultInstallation)

	defaultInstallation.Assess("dynakube applied", dynakube.ApplyDynakube(secretConfig.ApiUrl, &v1beta1.CloudNativeFullStackSpec{}, proxy))

	assessProxyStartup(defaultInstallation, proxy)

	assessDynakubeStartup(defaultInstallation)

	assessOneAgentsAreRunning(defaultInstallation)

	return defaultInstallation
}

func installAndDeploy(builder *features.FeatureBuilder, secretConfig secrets.Secret) {
	builder.Setup(secrets.ApplyDefault(secretConfig))
	builder.Setup(operator.InstallForKubernetes())
}

func installProxy(builder *features.FeatureBuilder) {
	builder.Setup(manifests.InstallFromFile("../testdata/activegate/proxy.yaml"))
}

func assessProxyStartup(builder *features.FeatureBuilder, proxy *v1beta1.DynaKubeProxy) {
	if proxy != nil {
		builder.Assess("proxy started", deployment.WaitFor(proxyDeployment, proxyNamespace))
	}
}

func assessDeployment(builder *features.FeatureBuilder) {
	builder.Assess("operator started", operator.WaitForDeployment())
	builder.Assess("webhook started", webhook.WaitForDeployment())
	builder.Assess("csi driver started", csi.WaitForDaemonset())
}

func assessDynakubeStartup(builder *features.FeatureBuilder) {
	builder.Assess("activegate started", WaitForStatefulSet())
	builder.Assess("oneagent started", oneagent.WaitForDaemonset())
	builder.Assess("dynakube phase changes to 'Running'", dynakube.WaitForDynakubePhase())
}

func assessOneAgentsAreRunning(builder *features.FeatureBuilder) {
	builder.Assess("osAgent can connect", oneagent.OsAgentsCanConnect())
}

func assessActiveGate(builder *features.FeatureBuilder) {
	builder.Assess("ActiveGate has required containers", checkIfAgHasContainers)
	builder.Assess("ActiveGate modules are active", checkActiveModules)
	builder.Assess("ActiveGate containers have mount points", checkMountPoints)
	builder.Assess("ActiveGate query via AG service", manifests.InstallFromFile("../testdata/activegate/curl-pod.yaml"))
	builder.Assess("ActiveGate query is completed", pod.WaitFor(curlPod, dynakube.DynatraceNamespace))
	builder.Assess("ActiveGate service is running", checkService)
}

func checkIfAgHasContainers(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
	resources := environmentConfig.Client().Resources()

	var pod corev1.Pod
	require.NoError(t, resources.WithNamespace(dynakube.DynatraceNamespace).Get(ctx, agPodName, agNamespace, &pod))

	require.NotNil(t, pod.Spec)
	require.NotEmpty(t, pod.Spec.InitContainers)
	require.NotEmpty(t, pod.Spec.Containers)

	isInitContainerUnknown(t, pod.Spec.InitContainers)
	isInitContainerMissing(t, pod.Spec.InitContainers)
	isContainerUnknown(t, pod.Spec.Containers)
	isContainerMissing(t, pod.Spec.Containers)

	return ctx
}

func checkActiveModules(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
	resources := environmentConfig.Client().Resources()

	var pod corev1.Pod
	require.NoError(t, resources.WithNamespace("dynatrace").Get(ctx, agPodName, agNamespace, &pod))

	clientset, err := kubernetes.NewForConfig(resources.GetConfig())
	require.NoError(t, err)

	logStream, err := clientset.CoreV1().Pods(agNamespace).GetLogs(agPodName, &corev1.PodLogOptions{
		Container: agContainerName,
	}).Stream(ctx)
	require.NoError(t, err)

	isModuleNotActive(t, logStream)

	return ctx
}

func checkMountPoints(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
	resources := environmentConfig.Client().Resources()

	var pod corev1.Pod
	require.NoError(t, resources.WithNamespace("dynatrace").Get(ctx, agPodName, agNamespace, &pod))

	for name, mountPoints := range agMounts {
		isMountPointMissing(t, environmentConfig, pod, name, mountPoints)
	}

	return ctx
}

func checkService(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
	resources := environmentConfig.Client().Resources()

	clientset, err := kubernetes.NewForConfig(resources.GetConfig())
	require.NoError(t, err)

	logStream, err := clientset.CoreV1().Pods(dynakube.DynatraceNamespace).GetLogs(curlPod, &corev1.PodLogOptions{
		Container: curlPod,
	}).Stream(ctx)
	require.NoError(t, err)

	logs.AssertLogContains(t, logStream, "RUNNING")

	return ctx
}

func isMountPointMissing(t *testing.T, environmentConfig *envconf.Config, podItem corev1.Pod, containerName string, mountPoints []string) {
	executionQuery := pod.NewExecutionQuery(podItem, containerName, "cat /proc/mounts")
	executionResult, err := executionQuery.Execute(environmentConfig.Client().RESTConfig())
	require.NoError(t, err)

	stdOut := executionResult.StdOut.String()
	stdErr := executionResult.StdErr.String()

	assert.Empty(t, stdErr)

	for _, mountPoint := range mountPoints {
		assert.True(t, strings.Contains(stdOut, mountPoint), "mount point not found: '"+mountPoint+"'")
	}
}

func isInitContainerUnknown(t *testing.T, podInitContainers []corev1.Container) {
	containers := initMap(&agInitContainers)

	for _, container := range podInitContainers {
		_, ok := containers[container.Name]
		assert.True(t, ok, "unknown init container: '"+container.Name+"'")
	}
}

func isInitContainerMissing(t *testing.T, podInitContainers []corev1.Container) {
	containers := initMap(&agInitContainers)

	markExistingContainers(&containers, podInitContainers)

	for name, container := range containers {
		assert.True(t, container, "init container is missing: '"+name+"'")
	}
}

func isContainerUnknown(t *testing.T, podContainers []corev1.Container) {
	containers := initMap(&agContainers)

	for _, container := range podContainers {
		_, ok := containers[container.Name]
		assert.True(t, ok, "unknown container: '"+container.Name+"'")
	}
}

func isContainerMissing(t *testing.T, podContainers []corev1.Container) {
	containers := initMap(&agContainers)

	markExistingContainers(&containers, podContainers)

	for name, container := range containers {
		assert.True(t, container, "container is missing: '"+name+"'")
	}
}

func isModuleNotActive(t *testing.T, logStream io.ReadCloser) {
	var expectedModules = []string{
		"kubernetes_monitoring",
		"extension_controller",
		"odin_collector",
		"metrics_ingest",
	}

	buffer := new(bytes.Buffer)
	_, err := io.Copy(buffer, logStream)
	require.NoError(t, err, "list of AG active modules not found")

	head := strings.SplitAfter(buffer.String(), "[<collector.modules>, ModulesManager] Modules:")
	require.Equal(t, 2, len(head), "list of AG active modules not found")

	tail := strings.SplitAfter(head[1], "Lifecycle listeners:")
	require.Equal(t, 2, len(head), "list of AG active modules not found")

	/*
		Expected log messages of the Gateway process:
			`Active:
				    kubernetes_monitoring"
				    extension_controller"
				    odin_collector"
				    metrics_ingest"
			Lifecycle listeners:`

		Warning: modules are printed in random order.
	*/
	for _, module := range expectedModules {
		assert.True(t, strings.Contains(tail[0], module), "ActiveGate module is not active: '"+module+"'")
	}
}

func markExistingContainers(containers *map[string]bool, podContainers []corev1.Container) {
	for _, container := range podContainers {
		if _, ok := (*containers)[container.Name]; ok {
			(*containers)[container.Name] = true
		}
	}
}

func initMap(srcMap *map[string]bool) map[string]bool {
	dstMap := make(map[string]bool)
	for k, v := range *srcMap {
		dstMap[k] = v
	}
	return dstMap
}

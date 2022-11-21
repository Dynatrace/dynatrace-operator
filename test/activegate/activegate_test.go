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
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/environment"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/manifests"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/logs"
	"github.com/Dynatrace/dynatrace-operator/test/oneagent"
	"github.com/Dynatrace/dynatrace-operator/test/operator"
	"github.com/Dynatrace/dynatrace-operator/test/proxy"
	"github.com/Dynatrace/dynatrace-operator/test/secrets"
	"github.com/Dynatrace/dynatrace-operator/test/webhook"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
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
			" /var/lib/dynatrace/gateway/config ",
			" /var/lib/dynatrace/secrets/tokens/tenant-token ",
			" /var/lib/dynatrace/secrets/tokens/auth-token ",
			" /var/lib/dynatrace/remotepluginmodule/log/extensions/eec ",
			" /var/lib/dynatrace/remotepluginmodule/log/extensions/statsd ",
			" /opt/dynatrace/gateway/jre/lib/security/cacerts ",
		},

		agEecContainerName: {
			" /var/lib/dynatrace/gateway/config ",
			" /var/lib/dynatrace/remotepluginmodule/log/extensions ",
			" /opt/dynatrace/remotepluginmodule/agent/datasources/statsd ",
			" /var/lib/dynatrace/remotepluginmodule/log/extensions/datasources-statsd ",
			" /var/lib/dynatrace/remotepluginmodule/agent/conf/runtime ",
			" /var/lib/dynatrace/remotepluginmodule/agent/runtime/datasources ",
		},
		agStatsdContainerName: {
			" /var/lib/dynatrace/remotepluginmodule/agent/runtime/datasources ",
			" /var/lib/dynatrace/remotepluginmodule/log/extensions/datasources-statsd ",
		},
	}
)

var testEnvironment env.Environment

func TestMain(m *testing.M) {
	testEnvironment = environment.Get()

	testEnvironment.BeforeEachTest(dynakube.DeleteIfExists(dynakube.NewBuilder().WithDefaultObjectMeta().Build()))
	testEnvironment.BeforeEachTest(oneagent.WaitForDaemonSetPodsDeletion())
	testEnvironment.BeforeEachTest(namespace.Recreate(namespace.NewBuilder(dynakube.Namespace).Build()))
	testEnvironment.BeforeEachTest(proxy.DeleteProxyIfExists())

	testEnvironment.AfterEachTest(dynakube.DeleteIfExists(dynakube.NewBuilder().WithDefaultObjectMeta().Build()))
	testEnvironment.AfterEachTest(oneagent.WaitForDaemonSetPodsDeletion())
	testEnvironment.AfterEachTest(namespace.Delete(dynakube.Namespace))
	testEnvironment.AfterEachTest(proxy.DeleteProxyIfExists())

	testEnvironment.Run(m)
}

func TestActiveGate(t *testing.T) {
	testEnvironment.Test(t, install(t, nil))
}

func TestActiveGateProxy(t *testing.T) {
	testEnvironment.Test(t, install(t, proxy.ProxySpec))
}

func install(t *testing.T, proxySpec *v1beta1.DynaKubeProxy) features.Feature {
	secretConfig, err := secrets.DefaultSingleTenant(afero.NewOsFs())

	require.NoError(t, err)

	defaultInstallation := features.New("capabilities")

	installAndDeploy(defaultInstallation, secretConfig)
	assessDeployment(defaultInstallation)

	proxy.InstallProxy(defaultInstallation, proxySpec)

	defaultInstallation.Assess("dynakube applied", dynakube.Apply(
		dynakube.NewBuilder().
			WithDefaultObjectMeta().
			WithActiveGate().
			WithDynakubeNamespaceSelector().
			ApiUrl(secretConfig.ApiUrl).
			CloudNative(&v1beta1.CloudNativeFullStackSpec{}).
			Proxy(proxySpec).
			Build()),
	)

	assessDynakubeStartup(defaultInstallation)
	assessOneAgentsAreRunning(defaultInstallation)
	assessActiveGate(defaultInstallation)

	return defaultInstallation.Feature()
}

func installAndDeploy(builder *features.FeatureBuilder, secretConfig secrets.Secret) {
	builder.Setup(secrets.ApplyDefault(secretConfig))
	builder.Setup(operator.InstallDynatrace(true))
}

func assessDeployment(builder *features.FeatureBuilder) {
	builder.Assess("operator started", operator.WaitForDeployment())
	builder.Assess("webhook started", webhook.WaitForDeployment())
	builder.Assess("csi driver started", csi.WaitForDaemonset())
}

func assessDynakubeStartup(builder *features.FeatureBuilder) {
	builder.Assess("oneagent started", oneagent.WaitForDaemonset())
	builder.Assess("dynakube phase changes to 'Running'", dynakube.WaitForDynakubePhase(
		dynakube.NewBuilder().WithDefaultObjectMeta().Build()))
}

func assessOneAgentsAreRunning(builder *features.FeatureBuilder) {
	builder.Assess("osAgent can connect", oneagent.OSAgentCanConnect())
}

func assessActiveGate(builder *features.FeatureBuilder) {
	builder.Assess("ActiveGate started", WaitForStatefulSet())
	builder.Assess("ActiveGate has required containers", checkIfAgHasContainers)
	builder.Assess("ActiveGate modules are active", checkActiveModules)
	builder.Assess("ActiveGate containers have mount points", checkMountPoints)
	builder.Assess("ActiveGate query via AG service", manifests.InstallFromFile("../testdata/activegate/curl-pod.yaml"))
	builder.Assess("ActiveGate query is completed", pod.WaitFor(curlPod, dynakube.Namespace))
	builder.Assess("ActiveGate service is running", checkService)
}

func checkIfAgHasContainers(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
	resources := environmentConfig.Client().Resources()

	var activeGatePod corev1.Pod
	require.NoError(t, resources.WithNamespace(dynakube.Namespace).Get(ctx, agPodName, agNamespace, &activeGatePod))

	require.NotNil(t, activeGatePod.Spec)
	require.NotEmpty(t, activeGatePod.Spec.InitContainers)
	require.NotEmpty(t, activeGatePod.Spec.Containers)

	assertInitContainerUnknown(t, activeGatePod.Spec.InitContainers)
	assertInitContainerMissing(t, activeGatePod.Spec.InitContainers)
	assertContainerUnknown(t, activeGatePod.Spec.Containers)
	assertContainerMissing(t, activeGatePod.Spec.Containers)

	return ctx
}

func checkActiveModules(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
	resources := environmentConfig.Client().Resources()

	var activeGatePod corev1.Pod
	require.NoError(t, resources.WithNamespace("dynatrace").Get(ctx, agPodName, agNamespace, &activeGatePod))

	clientset, err := kubernetes.NewForConfig(resources.GetConfig())
	require.NoError(t, err)

	logStream, err := clientset.CoreV1().Pods(agNamespace).GetLogs(agPodName, &corev1.PodLogOptions{
		Container: agContainerName,
	}).Stream(ctx)
	require.NoError(t, err)

	assertModuleNotActive(t, logStream)

	return ctx
}

func checkMountPoints(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
	resources := environmentConfig.Client().Resources()

	var activeGatePod corev1.Pod
	require.NoError(t, resources.WithNamespace("dynatrace").Get(ctx, agPodName, agNamespace, &activeGatePod))

	for name, mountPoints := range agMounts {
		assertMountPointMissing(t, environmentConfig, activeGatePod, name, mountPoints)
	}

	return ctx
}

func checkService(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
	resources := environmentConfig.Client().Resources()

	clientset, err := kubernetes.NewForConfig(resources.GetConfig())
	require.NoError(t, err)

	logStream, err := clientset.CoreV1().Pods(dynakube.Namespace).GetLogs(curlPod, &corev1.PodLogOptions{
		Container: curlPod,
	}).Stream(ctx)
	require.NoError(t, err)

	logs.AssertContains(t, logStream, "RUNNING")

	return ctx
}

func assertMountPointMissing(t *testing.T, environmentConfig *envconf.Config, podItem corev1.Pod, containerName string, mountPoints []string) {
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

func assertInitContainerUnknown(t *testing.T, podInitContainers []corev1.Container) {
	containers := initMap(&agInitContainers)

	for _, container := range podInitContainers {
		_, ok := containers[container.Name]
		assert.True(t, ok, "unknown init container: '"+container.Name+"'")
	}
}

func assertInitContainerMissing(t *testing.T, podInitContainers []corev1.Container) {
	containers := initMap(&agInitContainers)

	markExistingContainers(&containers, podInitContainers)

	for name, container := range containers {
		assert.True(t, container, "init container is missing: '"+name+"'")
	}
}

func assertContainerUnknown(t *testing.T, podContainers []corev1.Container) {
	containers := initMap(&agContainers)

	for _, container := range podContainers {
		_, ok := containers[container.Name]
		assert.True(t, ok, "unknown container: '"+container.Name+"'")
	}
}

func assertContainerMissing(t *testing.T, podContainers []corev1.Container) {
	containers := initMap(&agContainers)

	markExistingContainers(&containers, podContainers)

	for name, container := range containers {
		assert.True(t, container, "container is missing: '"+name+"'")
	}
}

func assertModuleNotActive(t *testing.T, logStream io.ReadCloser) {
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

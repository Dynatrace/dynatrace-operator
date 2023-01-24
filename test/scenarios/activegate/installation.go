package activegate

import (
	"bytes"
	"context"
	"io"
	"path"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/test/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/manifests"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/statefulset"
	"github.com/Dynatrace/dynatrace-operator/test/logs"
	"github.com/Dynatrace/dynatrace-operator/test/operator"
	"github.com/Dynatrace/dynatrace-operator/test/project"
	"github.com/Dynatrace/dynatrace-operator/test/proxy"
	"github.com/Dynatrace/dynatrace-operator/test/secrets"
	"github.com/Dynatrace/dynatrace-operator/test/shell"
	"github.com/Dynatrace/dynatrace-operator/test/webhook"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	agNamespace = "dynatrace"
	agPodName   = "dynakube-activegate-0"

	agContainerName = "activegate"

	curlPod = "curl"
)

var (
	agContainers = map[string]bool{
		agContainerName: false,
	}

	agInitContainers = map[string]bool{
		"certificate-loader": false,
	}

	agMounts = map[string][]string{
		agContainerName: {
			" /var/lib/dynatrace/secrets/tokens/tenant-token ",
			" /var/lib/dynatrace/secrets/tokens/auth-token ",
			" /opt/dynatrace/gateway/jre/lib/security/cacerts ",
		},
	}
)

func Install(t *testing.T, proxySpec *v1beta1.DynaKubeProxy) features.Feature {
	secretConfig, err := secrets.DefaultSingleTenant(afero.NewOsFs())

	require.NoError(t, err)

	defaultInstallation := features.New("capabilities")

	installAndDeploy(defaultInstallation, secretConfig)
	assessDeployment(defaultInstallation)

	proxy.InstallProxy(defaultInstallation, proxySpec)
	proxy.CutOffDynatraceNamespace(defaultInstallation, proxySpec)

	defaultInstallation.Assess("dynakube applied", dynakube.Apply(
		dynakube.NewBuilder().
			WithDefaultObjectMeta().
			WithActiveGate().
			WithDynakubeNamespaceSelector().
			ApiUrl(secretConfig.ApiUrl).
			Proxy(proxySpec).
			Build()),
	)

	assessDynakubeStartup(defaultInstallation)
	assessActiveGate(defaultInstallation, proxySpec)

	return defaultInstallation.Feature()
}

func installAndDeploy(builder *features.FeatureBuilder, secretConfig secrets.Secret) {
	builder.Setup(secrets.ApplyDefault(secretConfig))
	builder.Setup(operator.InstallViaMake(false))
}

func assessDeployment(builder *features.FeatureBuilder) {
	builder.Assess("operator started", operator.WaitForDeployment())
	builder.Assess("webhook started", webhook.WaitForDeployment())
}

func assessDynakubeStartup(builder *features.FeatureBuilder) {
	builder.Assess("dynakube phase changes to 'Running'", dynakube.WaitForDynakubePhase(
		dynakube.NewBuilder().WithDefaultObjectMeta().Build()))
}

func assessActiveGate(builder *features.FeatureBuilder, proxySpec *v1beta1.DynaKubeProxy) {
	builder.Assess("ActiveGate started", WaitForStatefulSet())
	builder.Assess("ActiveGate has required containers", checkIfAgHasContainers)
	builder.Assess("ActiveGate modules are active", checkActiveModules)
	if proxySpec != nil {
		builder.Assess("ActiveGate uses proxy", checkIfProxyUsed)
	}
	builder.Assess("ActiveGate containers have mount points", checkMountPoints)
	builder.Assess("ActiveGate query via AG service", manifests.InstallFromFile(path.Join(project.TestDataDir(), "activegate/curl-pod.yaml")))
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

	assertInitContainerKnown(t, activeGatePod.Spec.InitContainers)
	assertInitContainerExists(t, activeGatePod.Spec.InitContainers)
	assertContainersKnown(t, activeGatePod.Spec.Containers)
	assertContainersExist(t, activeGatePod.Spec.Containers)

	return ctx
}

func checkActiveModules(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
	log := readActiveGateLog(ctx, t, environmentConfig)
	assertExpectedModulesAreActive(t, log)
	return ctx
}

func checkIfProxyUsed(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
	log := readActiveGateLog(ctx, t, environmentConfig)
	assertProxyUsed(t, log)
	return ctx
}

func checkMountPoints(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
	resources := environmentConfig.Client().Resources()

	var activeGatePod corev1.Pod
	require.NoError(t, resources.WithNamespace("dynatrace").Get(ctx, agPodName, agNamespace, &activeGatePod))

	for name, mountPoints := range agMounts {
		assertMountPointsExist(t, environmentConfig, activeGatePod, name, mountPoints)
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

func assertMountPointsExist(t *testing.T, environmentConfig *envconf.Config, podItem corev1.Pod, containerName string, mountPoints []string) { //nolint:revive // argument-limit
	executionQuery := pod.NewExecutionQuery(podItem, containerName, shell.ReadFile("/proc/mounts")...)
	executionResult, err := executionQuery.Execute(environmentConfig.Client().RESTConfig())
	require.NoError(t, err)

	stdOut := executionResult.StdOut.String()
	stdErr := executionResult.StdErr.String()

	assert.Empty(t, stdErr)

	for _, mountPoint := range mountPoints {
		assert.True(t, strings.Contains(stdOut, mountPoint), "mount point not found: '"+mountPoint+"'")
		assert.Contains(t, stdOut, mountPoint, "mount point not found: '"+mountPoint+"'")
	}
}

func assertInitContainerKnown(t *testing.T, podInitContainers []corev1.Container) {
	containers := initMap(&agInitContainers)

	for _, container := range podInitContainers {
		_, ok := containers[container.Name]
		assert.True(t, ok, "unknown init container: '"+container.Name+"'")
	}
}

func assertInitContainerExists(t *testing.T, podInitContainers []corev1.Container) {
	containers := initMap(&agInitContainers)

	markExistingContainers(&containers, podInitContainers)

	for name, container := range containers {
		assert.True(t, container, "init container is missing: '"+name+"'")
	}
}

func assertContainersKnown(t *testing.T, podContainers []corev1.Container) {
	containers := initMap(&agContainers)

	for _, container := range podContainers {
		_, ok := containers[container.Name]
		assert.True(t, ok, "unknown container: '"+container.Name+"'")
	}
}

func assertContainersExist(t *testing.T, podContainers []corev1.Container) {
	containers := initMap(&agContainers)

	markExistingContainers(&containers, podContainers)

	for name, container := range containers {
		assert.True(t, container, "container is missing: '"+name+"'")
	}
}

func assertExpectedModulesAreActive(t *testing.T, log string) {
	var expectedModules = []string{
		"kubernetes_monitoring",
		"odin_collector",
		"metrics_ingest",
	}

	head := strings.SplitAfter(log, "[<collector.modules>, ModulesManager] Modules:")
	require.Equal(t, 2, len(head), "list of AG active modules not found")

	tail := strings.SplitAfter(head[1], "Lifecycle listeners:")
	require.Equal(t, 2, len(head), "list of AG active modules not found")

	/*
		Expected log messages of the Gateway process:
			`Active:
				    kubernetes_monitoring"
				    odin_collector"
				    metrics_ingest"
			Lifecycle listeners:`

		Warning: modules are printed in random order.
	*/
	for _, module := range expectedModules {
		assert.True(t, strings.Contains(tail[0], module), "ActiveGate module is not active: '"+module+"'")
	}
}

func assertProxyUsed(t *testing.T, log string) {
	assert.True(t, strings.Contains(log, "[HttpClientServiceImpl] Setup proxy server at: http://squid.proxy:3128"), "ActiveGate doesn't use proxy")
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

func WaitForStatefulSet() features.Func {
	return statefulset.WaitFor("dynakube-activegate", "dynatrace")
}

func readActiveGateLog(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) string {
	resources := environmentConfig.Client().Resources()

	var activeGatePod corev1.Pod
	require.NoError(t, resources.WithNamespace("dynatrace").Get(ctx, agPodName, agNamespace, &activeGatePod))

	clientset, err := kubernetes.NewForConfig(resources.GetConfig())
	require.NoError(t, err)

	logStream, err := clientset.CoreV1().Pods(agNamespace).GetLogs(agPodName, &corev1.PodLogOptions{
		Container: agContainerName,
	}).Stream(ctx)
	require.NoError(t, err)

	buffer := new(bytes.Buffer)
	_, err = io.Copy(buffer, logStream)
	require.NoError(t, err, "ActiveGate log not found")

	return buffer.String()
}

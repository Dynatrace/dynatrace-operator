//go:build e2e

package activegate

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/statefulset"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/proxy"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sampleapps"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/shell"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps/assess"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps/teardown"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

var (
	agContainers = map[string]bool{
		consts.ActiveGateContainerName: false,
	}

	agInitContainers = map[string]bool{
		"certificate-loader": false,
	}

	agMounts = map[string][]string{
		consts.ActiveGateContainerName: {
			" /var/lib/dynatrace/secrets/tokens/tenant-token ",
			" /var/lib/dynatrace/secrets/tokens/auth-token ",
			" /opt/dynatrace/gateway/jre/lib/security/cacerts ",
		},
	}
)

func Install(t *testing.T, proxySpec *dynatracev1beta1.DynaKubeProxy) features.Feature {
	builder := features.New("activegate-capabilities")
	secretConfig := tenant.GetSingleTenantSecret(t)
	testDynakube := dynakube.NewBuilder().
		WithDefaultObjectMeta().
		WithActiveGate().
		WithDynakubeNamespaceSelector().
		ApiUrl(secretConfig.ApiUrl).
		Proxy(proxySpec).
		Build()

	// Register operator install
	assess.InstallOperatorFromSource(builder, testDynakube)

	// Register proxy install and uninstall
	proxy.SetupProxyWithTeardown(builder, testDynakube)
	proxy.CutOffDynatraceNamespace(builder, proxySpec)

	// Register actual test
	assess.InstallDynakube(builder, &secretConfig, testDynakube)
	assessActiveGate(builder, &testDynakube)

	// Register operator + dynakube uninstall
	teardown.DeleteDynakube(builder, testDynakube)
	teardown.UninstallOperatorFromSource(builder, testDynakube)

	return builder.Feature()
}

func assessActiveGate(builder *features.FeatureBuilder, testDynakube *dynatracev1beta1.DynaKube) {
	builder.Assess("ActiveGate started", WaitForStatefulSet(testDynakube))
	builder.Assess("ActiveGate has required containers", checkIfAgHasContainers(testDynakube))
	builder.Assess("ActiveGate modules are active", checkActiveModules(testDynakube))
	if testDynakube.Spec.Proxy != nil {
		builder.Assess("ActiveGate uses proxy", checkIfProxyUsed(testDynakube))
	}
	builder.Assess("ActiveGate containers have mount points", checkMountPoints(testDynakube))
	builder.Assess("ActiveGate query via AG service", sampleapps.InstallActiveGateCurlPod(*testDynakube))
	builder.Assess("ActiveGate query is completed", sampleapps.WaitForActiveGateCurlPod(*testDynakube))
	builder.Assess("ActiveGate service is running", sampleapps.CheckActiveGateCurlResult(*testDynakube))
}

func checkIfAgHasContainers(testDynakube *dynatracev1beta1.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		resources := environmentConfig.Client().Resources()

		var activeGatePod corev1.Pod
		require.NoError(t, resources.WithNamespace(testDynakube.Namespace).Get(ctx, getActiveGatePodName(testDynakube), testDynakube.Namespace, &activeGatePod))

		require.NotNil(t, activeGatePod.Spec)
		require.NotEmpty(t, activeGatePod.Spec.InitContainers)
		require.NotEmpty(t, activeGatePod.Spec.Containers)

		assertInitContainerKnown(t, activeGatePod.Spec.InitContainers)
		assertInitContainerExists(t, activeGatePod.Spec.InitContainers)
		assertContainersKnown(t, activeGatePod.Spec.Containers)
		assertContainersExist(t, activeGatePod.Spec.Containers)

		return ctx
	}
}

func checkActiveModules(testDynakube *dynatracev1beta1.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		log := readActiveGateLog(ctx, t, environmentConfig, testDynakube)
		assertExpectedModulesAreActive(t, log)
		return ctx
	}
}

func checkIfProxyUsed(testDynakube *dynatracev1beta1.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		log := readActiveGateLog(ctx, t, environmentConfig, testDynakube)
		assertProxyUsed(t, log, testDynakube.Spec.Proxy.Value)
		return ctx
	}
}

func checkMountPoints(testDynakube *dynatracev1beta1.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		resources := environmentConfig.Client().Resources()

		var activeGatePod corev1.Pod
		require.NoError(t, resources.Get(ctx, getActiveGatePodName(testDynakube), testDynakube.Namespace, &activeGatePod))

		for name, mountPoints := range agMounts {
			assertMountPointsExist(ctx, t, resources, activeGatePod, name, mountPoints)
		}

		return ctx
	}
}

func assertMountPointsExist(ctx context.Context, t *testing.T, resources *resources.Resources, podItem corev1.Pod, containerName string, mountPoints []string) { //nolint:revive // argument-limit
	readFileCommand := shell.ReadFile("/proc/mounts")
	executionResult, err := pod.Exec(ctx, resources, podItem, containerName, readFileCommand...)
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

func assertProxyUsed(t *testing.T, log, proxyUrl string) {
	expectedLog := fmt.Sprintf("[HttpClientServiceImpl] Setup proxy server at: %s", proxyUrl)
	assert.True(t, strings.Contains(log, expectedLog), "ActiveGate doesn't use proxy")
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

func WaitForStatefulSet(testDynakube *dynatracev1beta1.DynaKube) features.Func {
	return statefulset.WaitFor(getActiveGateStateFulSetName(testDynakube), testDynakube.Namespace)
}

func readActiveGateLog(ctx context.Context, t *testing.T, environmentConfig *envconf.Config, testDynakube *dynatracev1beta1.DynaKube) string {
	resources := environmentConfig.Client().Resources()

	var activeGatePod corev1.Pod
	require.NoError(t, resources.WithNamespace(testDynakube.Namespace).Get(ctx, getActiveGatePodName(testDynakube), testDynakube.Namespace, &activeGatePod))

	clientset, err := kubernetes.NewForConfig(resources.GetConfig())
	require.NoError(t, err)

	logStream, err := clientset.CoreV1().Pods(testDynakube.Namespace).GetLogs(getActiveGatePodName(testDynakube), &corev1.PodLogOptions{
		Container: consts.ActiveGateContainerName,
	}).Stream(ctx)
	require.NoError(t, err)

	buffer := new(bytes.Buffer)
	_, err = io.Copy(buffer, logStream)
	require.NoError(t, err, "ActiveGate log not found")

	return buffer.String()
}

func getActiveGatePodName(testDynakube *dynatracev1beta1.DynaKube) string {
	return fmt.Sprintf("%s-0", getActiveGateStateFulSetName(testDynakube))
}

func getActiveGateStateFulSetName(testDynakube *dynatracev1beta1.DynaKube) string {
	return fmt.Sprintf("%s-activegate", testDynakube.Name)
}

//go:build e2e

package codemodules

import (
	"bytes"
	"context"
	"io"
	"os"
	"path"
	"strconv"
	"strings"
	"testing"
	"time"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	dtcsi "github.com/Dynatrace/dynatrace-operator/src/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/address"
	"github.com/Dynatrace/dynatrace-operator/src/webhook"
	"github.com/Dynatrace/dynatrace-operator/src/webhook/mutation/pod_mutator/oneagent_mutation"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/csi"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/istio"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/configmap"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/daemonset"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/deployment"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/proxy"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sampleapps"
	sample "github.com/Dynatrace/dynatrace-operator/test/helpers/sampleapps/base"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/setup"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/shell"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps/assess"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps/teardown"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/Dynatrace/dynatrace-operator/test/project"
	"github.com/Dynatrace/dynatrace-operator/test/scenarios/cloudnative"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	codeModulesVersion = "1.246.0.20220627-183412"
	codeModulesImage   = "quay.io/dynatrace/codemodules:" + codeModulesVersion
	diskUsageKiBDelta  = 100000

	dataPath                 = "/data/"
	provisionerContainerName = "provisioner"
)

func InstallFromImage(t *testing.T, istioEnabled bool) features.Feature {
	builder := features.New("codemodules injection")
	storageMap := make(map[string]int)
	secretConfigs := tenant.GetMultiTenantSecret(t)
	require.Len(t, secretConfigs, 2)

	dynakubeBuilder := dynakube.NewBuilder().
		WithDefaultObjectMeta().
		Name("cloudnative-codemodules").
		WithDynakubeNamespaceSelector().
		ApiUrl(secretConfigs[0].ApiUrl).
		CloudNative(codeModulesCloudNativeSpec())
	if istioEnabled {
		dynakubeBuilder = dynakubeBuilder.WithIstio()
	}
	cloudNativeDynakube := dynakubeBuilder.Build()

	dynakubeBuilder = dynakube.NewBuilder().
		WithDefaultObjectMeta().
		Name("app-codemodules").
		WithDynakubeNamespaceSelector().
		ApiUrl(secretConfigs[1].ApiUrl).
		ApplicationMonitoring(&dynatracev1beta1.ApplicationMonitoringSpec{
			AppInjectionSpec: *codeModulesAppInjectSpec(),
			UseCSIDriver:     address.Of(true),
		})
	if istioEnabled {
		dynakubeBuilder = dynakubeBuilder.WithIstio()
	}
	appDynakube := dynakubeBuilder.Build()

	namespaceBuilder := namespace.NewBuilder("codemodules-sample")
	labels := cloudNativeDynakube.NamespaceSelector().MatchLabels
	if istioEnabled {
		labels = kubeobjects.MergeMap(labels, istio.InjectionLabel)
	}
	sampleNamespace := namespaceBuilder.WithLabels(labels).Build()
	sampleApp := sampleapps.NewSampleDeployment(t, cloudNativeDynakube)
	sampleApp.WithNamespace(sampleNamespace)
	builder.Assess("create sample namespace", sampleApp.InstallNamespace())

	// Register operator install
	operatorNamespaceBuilder := namespace.NewBuilder(cloudNativeDynakube.Namespace)
	if istioEnabled {
		operatorNamespaceBuilder = operatorNamespaceBuilder.WithLabels(istio.InjectionLabel)
	}

	// Register dynakube install
	steps := setup.NewEnvironmentSetup(
		setup.CreateNamespaceWithoutTeardown(operatorNamespaceBuilder.Build()),
		setup.DeployOperatorViaMake(cloudNativeDynakube.NeedsCSIDriver()),
		setup.CreateDynakube(secretConfigs[0], cloudNativeDynakube),
	)
	steps.CreateSetupSteps(builder)

	// Register sample app install
	builder.Assess("install sample app", sampleApp.Install())

	// Register actual test
	cloudnative.AssessSampleInitContainers(builder, sampleApp)
	if istioEnabled {
		istio.AssessIstio(builder, cloudNativeDynakube, sampleApp)
	}

	builder.Assess("codemodules have been downloaded", imageHasBeenDownloaded(cloudNativeDynakube.Namespace))
	builder.Assess("checking storage used", measureDiskUsage(appDynakube.Namespace, storageMap))
	assess.InstallDynakube(builder, &secretConfigs[1], appDynakube)
	builder.Assess("storage size has not increased", diskUsageDoesNotIncrease(appDynakube.Namespace, storageMap))
	builder.Assess("volumes are mounted correctly", volumesAreMountedCorrectly(sampleApp))

	// Register sample, dynakube and operator uninstall
	builder.Teardown(sampleApp.UninstallNamespace())
	steps.CreateTeardownSteps(builder)

	return builder.Feature()
}

func withProxy(t *testing.T, proxySpec *dynatracev1beta1.DynaKubeProxy) features.Feature {
	builder := features.New("codemodules injection with proxy")
	secretConfigs := tenant.GetMultiTenantSecret(t)
	require.Len(t, secretConfigs, 2)

	dynakubeBuilder := dynakube.NewBuilder().
		WithDefaultObjectMeta().
		Name("cloudnative-codemodules-with-proxy").
		WithDynakubeNamespaceSelector().
		ApiUrl(secretConfigs[0].ApiUrl).
		CloudNative(codeModulesCloudNativeSpec()).
		WithIstio().
		Proxy(proxySpec)

	cloudNativeDynakube := dynakubeBuilder.Build()

	namespaceBuilder := namespace.NewBuilder("codemodules-sample-with-proxy")
	labels := cloudNativeDynakube.NamespaceSelector().MatchLabels
	labels = kubeobjects.MergeMap(labels, istio.InjectionLabel)

	sampleNamespace := namespaceBuilder.WithLabels(labels).Build()
	sampleApp := sampleapps.NewSampleDeployment(t, cloudNativeDynakube)
	sampleApp.WithNamespace(sampleNamespace)
	builder.Assess("create sample namespace", sampleApp.InstallNamespace())

	// Register operator install
	operatorNamespaceBuilder := namespace.NewBuilder(cloudNativeDynakube.Namespace)
	operatorNamespaceBuilder = operatorNamespaceBuilder.WithLabels(istio.InjectionLabel)

	steps := setup.NewEnvironmentSetup(
		setup.CreateNamespaceWithoutTeardown(operatorNamespaceBuilder.Build()),
		setup.DeployOperatorViaMake(cloudNativeDynakube.NeedsCSIDriver()))
	steps.CreateSetupSteps(builder)

	// Register proxy create and delete
	proxy.SetupProxyWithTeardown(t, builder, cloudNativeDynakube)
	proxy.CutOffDynatraceNamespace(builder, proxySpec)
	proxy.IsDynatraceNamespaceCutOff(builder, cloudNativeDynakube)

	// Register dynakube install
	assess.InstallDynakube(builder, &secretConfigs[0], cloudNativeDynakube)

	// Register sample app install
	builder.Assess("install sample app", sampleApp.Install())

	// Register actual test
	cloudnative.AssessSampleInitContainers(builder, sampleApp)
	istio.AssessIstio(builder, cloudNativeDynakube, sampleApp)

	builder.Assess("codemodules have been downloaded", imageHasBeenDownloaded(cloudNativeDynakube.Namespace))

	// Register sample, dynakube and operator uninstall
	builder.Teardown(sampleApp.UninstallNamespace())
	teardown.DeleteDynakube(builder, cloudNativeDynakube)
	steps.CreateTeardownSteps(builder)

	return builder.Feature()
}

func withProxyCA(t *testing.T, proxySpec *dynatracev1beta1.DynaKubeProxy) features.Feature {
	const configMapName = "proxy-ca"
	builder := features.New("codemodules injection with proxy and custom CA")
	secretConfigs := tenant.GetMultiTenantSecret(t)
	require.Len(t, secretConfigs, 2)

	dynakubeBuilder := dynakube.NewBuilder().
		WithDefaultObjectMeta().
		Name("cloudnative-codemodules-with-proxy-custom-ca").
		WithDynakubeNamespaceSelector().
		ApiUrl(secretConfigs[0].ApiUrl).
		CloudNative(codeModulesCloudNativeSpec()).
		WithCustomCAs(configMapName).
		WithIstio().
		Proxy(proxySpec)

	cloudNativeDynakube := dynakubeBuilder.Build()

	namespaceBuilder := namespace.NewBuilder("codemodules-sample-with-proxy-custom-ca")
	labels := cloudNativeDynakube.NamespaceSelector().MatchLabels
	labels = kubeobjects.MergeMap(labels, istio.InjectionLabel)

	sampleNamespace := namespaceBuilder.WithLabels(labels).Build()
	sampleApp := sampleapps.NewSampleDeployment(t, cloudNativeDynakube)
	sampleApp.WithNamespace(sampleNamespace)
	builder.Assess("create sample namespace", sampleApp.InstallNamespace())

	// Register operator install
	operatorNamespaceBuilder := namespace.NewBuilder(cloudNativeDynakube.Namespace)
	operatorNamespaceBuilder = operatorNamespaceBuilder.WithLabels(istio.InjectionLabel)

	// Need to create configmap
	builder.Assess("create operator namespace", namespace.Create(operatorNamespaceBuilder.Build()))
	// Add customCA config map
	trustedCa, _ := os.ReadFile(path.Join(project.TestDataDir(), "custom-cas/custom.pem"))
	configmapBuilder := configmap.NewBuilder(configMapName, cloudNativeDynakube.Namespace,
		map[string]string{dynatracev1beta1.TrustedCAKey: string(trustedCa)})
	builder.Assess("create trusted CAs config map", configmap.Create(configmapBuilder.Build()))

	steps := setup.NewEnvironmentSetup(
		setup.CreateNamespaceWithoutTeardown(operatorNamespaceBuilder.Build()),
		setup.DeployOperatorViaMake(cloudNativeDynakube.NeedsCSIDriver()))
	steps.CreateSetupSteps(builder)

	// Register proxy create and delete
	proxy.SetupProxyWithCustomCAandTeardown(t, builder, cloudNativeDynakube)
	proxy.CutOffDynatraceNamespace(builder, proxySpec)
	proxy.IsDynatraceNamespaceCutOff(builder, cloudNativeDynakube)

	// Register dynakube install
	assess.InstallDynakube(builder, &secretConfigs[0], cloudNativeDynakube)

	// Register sample app install
	builder.Assess("install sample app", sampleApp.Install())

	// Register actual test
	cloudnative.AssessSampleInitContainers(builder, sampleApp)
	istio.AssessIstio(builder, cloudNativeDynakube, sampleApp)

	builder.Assess("codemodules have been downloaded", imageHasBeenDownloaded(cloudNativeDynakube.Namespace))

	// Register sample, dynakube and operator uninstall
	builder.Teardown(sampleApp.UninstallNamespace())
	teardown.DeleteDynakube(builder, cloudNativeDynakube)
	steps.CreateTeardownSteps(builder)

	return builder.Feature()
}

func codeModulesCloudNativeSpec() *dynatracev1beta1.CloudNativeFullStackSpec {
	return &dynatracev1beta1.CloudNativeFullStackSpec{
		HostInjectSpec: dynatracev1beta1.HostInjectSpec{
			Args: []string{"INTERNAL_OVERRIDE_CHECKS=downgrade"},
		},
		AppInjectionSpec: *codeModulesAppInjectSpec(),
	}
}

func codeModulesAppInjectSpec() *dynatracev1beta1.AppInjectionSpec {
	return &dynatracev1beta1.AppInjectionSpec{
		CodeModulesImage: codeModulesImage,
	}
}

func imageHasBeenDownloaded(namespace string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resource := envConfig.Client().Resources()
		clientset, err := kubernetes.NewForConfig(resource.GetConfig())
		require.NoError(t, err)

		err = daemonset.NewQuery(ctx, resource, client.ObjectKey{
			Name:      csi.DaemonSetName,
			Namespace: namespace,
		}).ForEachPod(func(podItem corev1.Pod) {
			err = wait.For(func(ctx context.Context) (done bool, err error) {
				logStream, err := clientset.CoreV1().Pods(podItem.Namespace).GetLogs(podItem.Name, &corev1.PodLogOptions{
					Container: provisionerContainerName,
				}).Stream(ctx)
				require.NoError(t, err)
				buffer := new(bytes.Buffer)
				_, err = io.Copy(buffer, logStream)
				isNew := strings.Contains(buffer.String(), "Installed agent version: "+codeModulesImage)
				isOld := strings.Contains(buffer.String(), "agent already installed")
				t.Logf("wait for Installed agent version in %s", podItem.Name)
				return isNew || isOld, err
			}, wait.WithTimeout(time.Minute*5))
			require.NoError(t, err)

			listCommand := shell.ListDirectory(dataPath)
			result, err := pod.Exec(ctx, resource, podItem, provisionerContainerName, listCommand...)

			require.NoError(t, err)
			assert.Contains(t, result.StdOut.String(), dtcsi.SharedAgentBinDir)
		})

		require.NoError(t, err)

		return ctx
	}
}

func measureDiskUsage(namespace string, storageMap map[string]int) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resource := envConfig.Client().Resources()
		err := daemonset.NewQuery(ctx, resource, client.ObjectKey{
			Name:      csi.DaemonSetName,
			Namespace: namespace,
		}).ForEachPod(func(podItem corev1.Pod) {
			diskUsage := getDiskUsage(ctx, t, envConfig.Client().Resources(), podItem, provisionerContainerName, dataPath)
			storageMap[podItem.Name] = diskUsage
		})
		require.NoError(t, err)
		return ctx
	}
}

func diskUsageDoesNotIncrease(namespace string, storageMap map[string]int) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resource := envConfig.Client().Resources()
		err := daemonset.NewQuery(ctx, resource, client.ObjectKey{
			Name:      csi.DaemonSetName,
			Namespace: namespace,
		}).ForEachPod(func(podItem corev1.Pod) {
			diskUsage := getDiskUsage(ctx, t, envConfig.Client().Resources(), podItem, provisionerContainerName, dataPath)
			assert.InDelta(t, storageMap[podItem.Name], diskUsage, diskUsageKiBDelta)
		})
		require.NoError(t, err)

		return ctx
	}
}

func getDiskUsage(ctx context.Context, t *testing.T, resource *resources.Resources, podItem corev1.Pod, containerName, path string) int { //nolint:revive
	diskUsageCommand := shell.Shell(
		shell.Pipe(
			shell.DiskUsageWithTotal(path),
			shell.FilterLastLineOnly(),
		),
	)
	result, err := pod.Exec(ctx, resource, podItem, containerName, diskUsageCommand...)
	require.NoError(t, err)

	diskUsage, err := strconv.Atoi(strings.Split(result.StdOut.String(), "\t")[0])
	require.NoError(t, err)

	return diskUsage
}

func volumesAreMountedCorrectly(sampleApp sample.App) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resource := envConfig.Client().Resources()
		err := deployment.NewQuery(ctx, resource, client.ObjectKey{
			Name:      sampleApp.Name(),
			Namespace: sampleApp.Namespace().Name,
		}).ForEachPod(func(podItem corev1.Pod) {
			volumes := podItem.Spec.Volumes
			volumeMounts := podItem.Spec.Containers[0].VolumeMounts

			assert.True(t, isVolumeAttached(t, volumes, oneagent_mutation.OneAgentBinVolumeName))
			assert.True(t, isVolumeMounted(t, volumeMounts, oneagent_mutation.OneAgentBinVolumeName))

			listCommand := shell.ListDirectory(webhook.DefaultInstallPath)
			executionResult, err := pod.Exec(ctx, resource, podItem, sampleApp.ContainerName(), listCommand...)

			require.NoError(t, err)
			assert.NotEmpty(t, executionResult.StdOut.String())

			diskUsage := getDiskUsage(ctx, t, envConfig.Client().Resources(), podItem, sampleApp.ContainerName(), webhook.DefaultInstallPath)
			assert.Greater(t, diskUsage, 0)
		})

		require.NoError(t, err)
		return ctx
	}
}

func isVolumeMounted(t *testing.T, volumeMounts []corev1.VolumeMount, volumeMountName string) bool {
	result := false
	for _, volumeMount := range volumeMounts {
		if volumeMount.Name == volumeMountName {
			result = true

			assert.Equal(t, webhook.DefaultInstallPath, volumeMount.MountPath)
			assert.False(t, volumeMount.ReadOnly)
		}
	}
	return result
}

func isVolumeAttached(t *testing.T, volumes []corev1.Volume, volumeName string) bool {
	result := false
	for _, volume := range volumes {
		if volume.Name == volumeName {
			result = true

			assert.NotNil(t, volume.CSI)
			assert.Equal(t, dtcsi.DriverName, volume.CSI.Driver)

			if volume.CSI.ReadOnly != nil {
				assert.False(t, *volume.CSI.ReadOnly)
			}
		}
	}
	return result
}

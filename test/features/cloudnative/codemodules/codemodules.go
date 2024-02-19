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

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/address"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod_mutator/oneagent_mutation"
	"github.com/Dynatrace/dynatrace-operator/test/features/cloudnative"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/csi"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/istio"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/configmap"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/daemonset"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/deployment"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/proxy"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sample"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/shell"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/Dynatrace/dynatrace-operator/test/project"
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

// Verification that the storage in the CSI driver directory does not increase when
// there are multiple tenants and pods which are monitored.
func InstallFromImage(t *testing.T) features.Feature {
	builder := features.New("cloudnative codemodules injection from image")
	builder.WithLabel("name", "cloudnative-codemodules-image")
	storageMap := make(map[string]int)
	secretConfigs := tenant.GetMultiTenantSecret(t)
	require.Len(t, secretConfigs, 2)

	cloudNativeDynakube := *dynakube.New(
		dynakube.WithName("cloudnative-codemodules"),
		dynakube.WithNameBasedNamespaceSelector(),
		dynakube.WithApiUrl(secretConfigs[0].ApiUrl),
		dynakube.WithCloudNativeSpec(codeModulesCloudNativeSpec()),
	)

	appDynakube := *dynakube.New(
		dynakube.WithName("app-codemodules"),
		dynakube.WithNameBasedNamespaceSelector(),
		dynakube.WithApiUrl(secretConfigs[1].ApiUrl),
		dynakube.WithApplicationMonitoringSpec(&dynatracev1beta1.ApplicationMonitoringSpec{
			AppInjectionSpec: *codeModulesAppInjectSpec(),
			UseCSIDriver:     address.Of(true),
		}),
	)

	labels := cloudNativeDynakube.NamespaceSelector().MatchLabels
	sampleNamespace := *namespace.New("codemodules-sample", namespace.WithLabels(labels))

	sampleApp := sample.NewApp(t, &cloudNativeDynakube,
		sample.AsDeployment(),
		sample.WithNamespace(sampleNamespace),
	)

	builder.Assess("create sample namespace", sampleApp.InstallNamespace())

	// Register dynakube install
	dynakube.Install(builder, helpers.LevelAssess, &secretConfigs[0], cloudNativeDynakube)

	// Register sample app install
	builder.Assess("install sample app", sampleApp.Install())

	// Register actual test
	cloudnative.AssessSampleInitContainers(builder, sampleApp)

	builder.Assess("codemodules have been downloaded", imageHasBeenDownloaded(cloudNativeDynakube.Namespace))
	builder.Assess("checking storage used", measureDiskUsage(appDynakube.Namespace, storageMap))
	dynakube.Install(builder, helpers.LevelAssess, &secretConfigs[1], appDynakube)
	builder.Assess("storage size has not increased", diskUsageDoesNotIncrease(appDynakube.Namespace, storageMap))
	builder.Assess("volumes are mounted correctly", volumesAreMountedCorrectly(*sampleApp))

	// Register sample, dynakube and operator uninstall
	builder.Teardown(sampleApp.Uninstall())
	dynakube.Delete(builder, helpers.LevelTeardown, cloudNativeDynakube)
	dynakube.Delete(builder, helpers.LevelTeardown, appDynakube)

	return builder.Feature()
}

const (
	httpsProxy = "https_proxy"
)

// Prerequisites: istio service mesh
//
// Setup: CloudNative deployment with CSI driver
//
// Verification that the operator and all deployed OneAgents are able to communicate
// over a http proxy.
//
// Connectivity in the dynatrace namespace and sample application namespace is restricted to
// the local cluster. Sample application is installed. The test checks if DT_PROXY environment
// variable is defined in the *dynakube-oneagent* container and the *application container*.
func WithProxy(t *testing.T, proxySpec *dynatracev1beta1.DynaKubeProxy) features.Feature {
	builder := features.New("codemodules injection with proxy")
	builder.WithLabel("name", "codemodules-with-proxy")
	secretConfigs := tenant.GetMultiTenantSecret(t)
	require.Len(t, secretConfigs, 2)

	cloudNativeDynakube := *dynakube.New(
		dynakube.WithName("codemodules-with-proxy"),
		dynakube.WithApiUrl(secretConfigs[0].ApiUrl),
		dynakube.WithCloudNativeSpec(codeModulesCloudNativeSpec()),
		dynakube.WithIstioIntegration(),
		dynakube.WithProxy(proxySpec),
	)

	sampleNamespace := *namespace.New("codemodules-sample-with-proxy", namespace.WithIstio())
	sampleApp := sample.NewApp(t, &cloudNativeDynakube,
		sample.AsDeployment(),
		sample.WithNamespace(sampleNamespace),
	)

	builder.Assess("create sample namespace", sampleApp.InstallNamespace())

	// Register proxy create and delete
	proxy.SetupProxyWithTeardown(t, builder, cloudNativeDynakube)
	proxy.CutOffDynatraceNamespace(builder, proxySpec)
	proxy.IsDynatraceNamespaceCutOff(builder, cloudNativeDynakube)

	// Register dynakube install
	dynakube.Install(builder, helpers.LevelAssess, &secretConfigs[0], cloudNativeDynakube)

	// Register sample app install
	builder.Assess("install sample app", sampleApp.Install())

	// Register actual test
	cloudnative.AssessSampleInitContainers(builder, sampleApp)
	istio.AssessIstio(builder, cloudNativeDynakube, *sampleApp)
	builder.Assess("codemodules have been downloaded", imageHasBeenDownloaded(cloudNativeDynakube.Namespace))

	builder.Assess("check env variables of oneagent pods", checkOneAgentEnvVars(cloudNativeDynakube))
	builder.Assess("check proxy settings in ruxitagentproc.conf", proxy.CheckRuxitAgentProcFileHasProxySetting(*sampleApp, proxySpec))

	// Register sample, dynakube and operator uninstall
	builder.Teardown(sampleApp.Uninstall())
	dynakube.Delete(builder, helpers.LevelTeardown, cloudNativeDynakube)

	return builder.Feature()
}

func WithProxyCA(t *testing.T, proxySpec *dynatracev1beta1.DynaKubeProxy) features.Feature {
	const configMapName = "proxy-ca"
	builder := features.New("codemodules injection with proxy and custom CA")
	builder.WithLabel("name", "codemodules-with-proxy-custom-ca")
	secretConfigs := tenant.GetMultiTenantSecret(t)
	require.Len(t, secretConfigs, 2)

	cloudNativeDynakube := *dynakube.New(
		dynakube.WithName("codemodules-with-proxy-custom-ca"),
		dynakube.WithApiUrl(secretConfigs[0].ApiUrl),
		dynakube.WithCloudNativeSpec(codeModulesCloudNativeSpec()),
		dynakube.WithCustomCAs(configMapName),
		dynakube.WithIstioIntegration(),
		dynakube.WithProxy(proxySpec),
	)

	sampleNamespace := *namespace.New("codemodules-sample-with-proxy-custom-ca", namespace.WithIstio())
	sampleApp := sample.NewApp(t, &cloudNativeDynakube,
		sample.AsDeployment(),
		sample.WithNamespace(sampleNamespace),
	)

	builder.Assess("create sample namespace", sampleApp.InstallNamespace())

	// Add customCA config map
	trustedCa, _ := os.ReadFile(path.Join(project.TestDataDir(), "custom-cas/custom.pem"))
	caConfigMap := configmap.New(configMapName, cloudNativeDynakube.Namespace,
		map[string]string{dynatracev1beta1.TrustedCAKey: string(trustedCa)})
	builder.Assess("create trusted CAs config map", configmap.Create(caConfigMap))

	// Register proxy create and delete
	proxy.SetupProxyWithCustomCAandTeardown(t, builder, cloudNativeDynakube)
	proxy.CutOffDynatraceNamespace(builder, proxySpec)
	proxy.IsDynatraceNamespaceCutOff(builder, cloudNativeDynakube)

	// Register dynakube install
	dynakube.Install(builder, helpers.LevelAssess, &secretConfigs[0], cloudNativeDynakube)

	// Register sample app install
	builder.Assess("install sample app", sampleApp.Install())

	// Register actual test
	cloudnative.AssessSampleInitContainers(builder, sampleApp)
	istio.AssessIstio(builder, cloudNativeDynakube, *sampleApp)

	builder.Assess("codemodules have been downloaded", imageHasBeenDownloaded(cloudNativeDynakube.Namespace))

	// Register sample, dynakube and operator uninstall
	builder.Teardown(sampleApp.Uninstall())
	dynakube.Delete(builder, helpers.LevelTeardown, cloudNativeDynakube)

	return builder.Feature()
}

func codeModulesCloudNativeSpec() *dynatracev1beta1.CloudNativeFullStackSpec {
	return &dynatracev1beta1.CloudNativeFullStackSpec{
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
			Namespace: sampleApp.Namespace(),
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

			require.NotNil(t, volume.CSI)
			assert.Equal(t, dtcsi.DriverName, volume.CSI.Driver)

			if volume.CSI.ReadOnly != nil {
				assert.False(t, *volume.CSI.ReadOnly)
			}
		}
	}

	return result
}

func checkOneAgentEnvVars(dynakube dynatracev1beta1.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()
		err := daemonset.NewQuery(ctx, resources, client.ObjectKey{
			Name:      dynakube.OneAgentDaemonsetName(),
			Namespace: dynakube.Namespace,
		}).ForEachPod(func(podItem corev1.Pod) {
			require.NotNil(t, podItem)
			require.NotNil(t, podItem.Spec)

			checkEnvVarsInContainer(t, podItem, dynakube.OneAgentDaemonsetName(), httpsProxy)
		})

		require.NoError(t, err)

		return ctx
	}
}

func checkEnvVarsInContainer(t *testing.T, podItem corev1.Pod, containerName string, envVar string) {
	for _, container := range podItem.Spec.Containers {
		if container.Name == containerName {
			require.NotNil(t, container.Env)
			require.True(t, env.IsIn(container.Env, envVar))
			for _, env := range container.Env {
				if env.Name == envVar {
					require.NotNil(t, env.Value)
				}
			}
		}
	}
}

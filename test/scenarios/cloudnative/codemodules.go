//go:build e2e

package cloudnative

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	dtcsi "github.com/Dynatrace/dynatrace-operator/src/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/address"
	"github.com/Dynatrace/dynatrace-operator/src/webhook"
	"github.com/Dynatrace/dynatrace-operator/src/webhook/mutation/pod_mutator/oneagent_mutation"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/csi"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/istio"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/deployment"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sampleapps"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/shell"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps/assess"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps/teardown"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	codeModulesVersion     = "1.246.0.20220627-183412"
	codeModulesImage       = "quay.io/dynatrace/codemodules:" + codeModulesVersion
	codeModulesImageDigest = "7ece13a07a20c77a31cc36906a10ebc90bd47970905ee61e8ed491b7f4c5d62f"

	dataPath                 = "/data/"
	provisionerContainerName = "provisioner"
)

type manifest struct {
	Version string `json:"version,omitempty"`
}

func CodeModules(t *testing.T, istioEnabled bool) features.Feature {
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
	if istioEnabled {
		namespaceBuilder = namespaceBuilder.WithLabels(istio.IstioLabel)
	}
	sampleNamespace := namespaceBuilder.WithLabels(cloudNativeDynakube.NamespaceSelector().MatchLabels).Build()
	sampleApp := sampleapps.NewSampleDeployment(t, cloudNativeDynakube)
	sampleApp.WithNamespace(sampleNamespace)

	// Register operator install
	assess.InstallOperatorFromSource(builder, cloudNativeDynakube)

	// Register actual test
	assess.InstallDynakube(builder, &secretConfigs[0], cloudNativeDynakube)
	builder.Assess("install sample app", sampleApp.Install())
	assessSampleInitContainers(builder, sampleApp)
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
	teardown.UninstallDynatrace(builder, cloudNativeDynakube)

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
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		resource := environmentConfig.Client().Resources()
		restConfig := environmentConfig.Client().RESTConfig()

		err := csi.ForEachPod(ctx, resource, namespace, func(podItem corev1.Pod) {
			var result *pod.ExecutionResult
			result, err := pod.
				NewExecutionQuery(podItem, provisionerContainerName, shell.ListDirectory(dataPath)...).
				Execute(restConfig)

			require.NoError(t, err)
			assert.Contains(t, result.StdOut.String(), dtcsi.SharedAgentBinDir)

			result, err = pod.
				NewExecutionQuery(podItem, provisionerContainerName, shell.Shell(shell.ReadFile(getManifestPath()))...).
				Execute(restConfig)

			require.NoError(t, err)

			var codeModulesManifest manifest
			err = json.Unmarshal(result.StdOut.Bytes(), &codeModulesManifest)
			if err != nil {
				err = errors.WithMessagef(err, "json:\n%s", result.StdOut)
			}
			require.NoError(t, err)

			assert.Equal(t, codeModulesVersion, codeModulesManifest.Version)
		})

		require.NoError(t, err)

		return ctx
	}
}

func measureDiskUsage(namespace string, storageMap map[string]int) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		resource := environmentConfig.Client().Resources()
		restConfig := environmentConfig.Client().RESTConfig()
		err := csi.ForEachPod(ctx, resource, namespace, func(podItem corev1.Pod) {
			var result *pod.ExecutionResult
			result, err := pod.
				NewExecutionQuery(podItem, provisionerContainerName, shell.Shell(shell.Pipe(
					shell.DiskUsageWithTotal(dataPath),
					shell.FilterLastLineOnly()))...).
				Execute(restConfig)

			require.NoError(t, err)

			diskUsage, err := strconv.Atoi(strings.Split(result.StdOut.String(), "\t")[0])

			require.NoError(t, err)

			storageMap[podItem.Name] = diskUsage
		})
		require.NoError(t, err)
		return ctx
	}
}

func diskUsageDoesNotIncrease(namespace string, storageMap map[string]int) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		resource := environmentConfig.Client().Resources()
		restConfig := environmentConfig.Client().RESTConfig()
		err := csi.ForEachPod(ctx, resource, namespace, func(podItem corev1.Pod) {
			var result *pod.ExecutionResult
			result, err := pod.
				NewExecutionQuery(podItem, provisionerContainerName, shell.Shell(shell.Pipe(
					shell.DiskUsageWithTotal(dataPath),
					shell.FilterLastLineOnly()))...).
				Execute(restConfig)

			require.NoError(t, err)

			diskUsage, err := strconv.Atoi(strings.Split(result.StdOut.String(), "\t")[0])

			require.NoError(t, err)
			// Dividing it by 1000 so the sizes do not need to be exactly the same down to the byte
			assert.Equal(t, storageMap[podItem.Name]/1000, diskUsage/1000)
		})
		require.NoError(t, err)

		return ctx
	}
}

func volumesAreMountedCorrectly(sampleApp sampleapps.SampleApp) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		resource := environmentConfig.Client().Resources()
		err := deployment.NewQuery(ctx, resource, client.ObjectKey{
			Name:      sampleApp.Name(),
			Namespace: sampleApp.Namespace().Name,
		}).ForEachPod(func(podItem corev1.Pod) {
			volumes := podItem.Spec.Volumes
			volumeMounts := podItem.Spec.Containers[0].VolumeMounts

			assert.True(t, isVolumeAttached(t, volumes, oneagent_mutation.OneAgentBinVolumeName))
			assert.True(t, isVolumeMounted(t, volumeMounts, oneagent_mutation.OneAgentBinVolumeName))

			executionResult, err := pod.
				NewExecutionQuery(podItem, sampleApp.ContainerName(), shell.ListDirectory(webhook.DefaultInstallPath)...).
				Execute(environmentConfig.Client().RESTConfig())

			require.NoError(t, err)
			assert.NotEmpty(t, executionResult.StdOut.String())

			executionResult, err = pod.
				NewExecutionQuery(podItem, sampleApp.ContainerName(), shell.Shell(shell.Pipe(
					shell.DiskUsageWithTotal(webhook.DefaultInstallPath),
					shell.FilterLastLineOnly()))...).
				Execute(environmentConfig.Client().RESTConfig())

			require.NoError(t, err)
			require.Contains(t, executionResult.StdOut.String(), "total")

			diskUsage, err := strconv.Atoi(strings.Split(executionResult.StdOut.String(), "\t")[0])

			require.NoError(t, err)
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

func getManifestPath() string {
	return "/data/codemodules/" + codeModulesImageDigest + "/manifest.json"
}

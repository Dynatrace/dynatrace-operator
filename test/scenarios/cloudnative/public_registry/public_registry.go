//go:build e2e

package public_registry

import (
	"context"
	"testing"
	"time"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/csi"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/oneagent"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/daemonset"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/logs"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps/assess"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps/teardown"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	publicRegistrySource     = dynatracev1beta1.VersionSource("public-registry")
	provisionerContainerName = "provisioner"
	customPullSecretName     = "devregistry"
)

var publicRegistryFeatureFlag = map[string]string{dynatracev1beta1.AnnotationFeaturePublicRegistry: "true"}

func publicRegistry(t *testing.T) features.Feature {
	builder := features.New("cloudnative with public registry feature enabled")
	secretConfig := tenant.GetSingleTenantSecret(t)

	dynakubeBuilder := dynakube.NewBuilder().
		WithDefaultObjectMeta().
		WithCustomPullSecret(customPullSecretName).
		WithDynakubeNamespaceSelector().
		WithAnnotations(publicRegistryFeatureFlag).
		WithActiveGate().
		ApiUrl(secretConfig.ApiUrl).
		CloudNative(&dynatracev1beta1.CloudNativeFullStackSpec{})
	testDynakube := dynakubeBuilder.Build()

	// Register operator + dynakube install
	assess.InstallDynatrace(builder, &secretConfig, testDynakube)

	builder.Assess("check dynakube status", checkDynakubeStatus(testDynakube))
	builder.Assess("check whether public registry images are used", checkPublicRegistryUsage(testDynakube))
	builder.Assess("check whether correct image has been downloaded", checkCSIDriver(testDynakube))

	// Register dynakube and operator uninstall
	teardown.UninstallDynatrace(builder, testDynakube)

	return builder.Feature()
}

func checkDynakubeStatus(dynakube dynatracev1beta1.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		resources := environmentConfig.Client().Resources()
		var dk dynatracev1beta1.DynaKube

		err := dynatracev1beta1.AddToScheme(resources.GetScheme())
		require.NoError(t, err)

		err = resources.Get(ctx, dynakube.Name, dynakube.Namespace, &dk)
		require.NoError(t, err)

		require.NotNil(t, dk.Status.OneAgent)
		require.NotEmpty(t, dk.Status.OneAgent.VersionStatus.ImageID)
		require.NotEmpty(t, dk.Status.OneAgent.VersionStatus.Source)
		require.NotNil(t, dk.Status.OneAgent.VersionStatus.LastProbeTimestamp)

		require.NotNil(t, dk.Status.CodeModules)
		require.NotEmpty(t, dk.Status.CodeModules.VersionStatus.ImageID)
		require.NotEmpty(t, dk.Status.CodeModules.VersionStatus.Source)
		require.NotNil(t, dk.Status.CodeModules.VersionStatus.LastProbeTimestamp)

		require.NotNil(t, dk.Status.ActiveGate)
		require.NotEmpty(t, dk.Status.ActiveGate.VersionStatus.ImageID)
		require.NotEmpty(t, dk.Status.ActiveGate.VersionStatus.Source)
		require.NotNil(t, dk.Status.ActiveGate.VersionStatus.LastProbeTimestamp)

		require.Equal(t, dk.Status.OneAgent.VersionStatus.Source, publicRegistrySource)
		require.Equal(t, dk.Status.CodeModules.VersionStatus.Source, publicRegistrySource)
		require.Equal(t, dk.Status.ActiveGate.VersionStatus.Source, publicRegistrySource)

		return ctx
	}
}

func checkPublicRegistryUsage(dynakube dynatracev1beta1.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		resources := environmentConfig.Client().Resources()
		var dk dynatracev1beta1.DynaKube

		err := dynatracev1beta1.AddToScheme(resources.GetScheme())
		require.NoError(t, err)

		err = resources.Get(ctx, dynakube.Name, dynakube.Namespace, &dk)
		require.NoError(t, err)

		oneAgentDaemonSet, err := oneagent.Get(ctx, resources, dynakube)
		require.NoError(t, err)

		require.Equal(t, dk.Status.OneAgent.ImageID, oneAgentDaemonSet.Spec.Template.Spec.Containers[0].Image)

		var activeGateStateFulSet v1.StatefulSet
		require.NoError(t, resources.Get(ctx, dynakube.Name+"-activegate", dynakube.Namespace, &activeGateStateFulSet))

		require.Equal(t, dk.Status.ActiveGate.ImageID, activeGateStateFulSet.Spec.Template.Spec.Containers[0].Image)

		return ctx
	}
}

func checkCSIDriver(dynakube dynatracev1beta1.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		resources := environmentConfig.Client().Resources()
		clientset, err := kubernetes.NewForConfig(resources.GetConfig())

		err = daemonset.NewQuery(ctx, resources, client.ObjectKey{
			Name:      csi.DaemonSetName,
			Namespace: dynakube.Namespace,
		}).ForEachPod(func(podItem corev1.Pod) {
			err = wait.For(func() (done bool, err error) {
				logStream, err := clientset.CoreV1().Pods(podItem.Namespace).GetLogs(podItem.Name, &corev1.PodLogOptions{
					Container: provisionerContainerName,
				}).Stream(ctx)
				require.NoError(t, err)
				return logs.Contains(t, logStream, "Installed agent version: "+dynakube.Status.CodeModules.ImageID), err
			}, wait.WithTimeout(time.Minute*5))
			require.NoError(t, err)
		})

		require.NoError(t, err)

		return ctx
	}
}

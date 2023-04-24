//go:build e2e

package applicationmonitoring

import (
	"context"
	"path"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/address"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/manifests"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sampleapps"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps/assess"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps/teardown"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/Dynatrace/dynatrace-operator/test/project"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	sampleAppNamespace = "appmon-sample"
)

func withoutCSIDriver(t *testing.T) features.Feature {
	builder := features.New("application monitoring without csi driver enabled")
	secretConfig := tenant.GetSingleTenantSecret(t)
	defaultDynakubeName := "dynakube"
	dynakubeBuilder := dynakube.NewBuilder().
		WithDefaultObjectMeta().
		Name(defaultDynakubeName).
		ApiUrl(secretConfig.ApiUrl).
		NamespaceSelector(metav1.LabelSelector{
			MatchLabels: map[string]string{
				"inject": defaultDynakubeName,
			},
		}).
		ApplicationMonitoring(&dynatracev1beta1.ApplicationMonitoringSpec{
			UseCSIDriver: address.Of(false),
		})

	appOnlyDynakube := dynakubeBuilder.Build()

	dynakubeBuilder = dynakube.NewBuilder().
		WithDefaultObjectMeta().
		Name("appmon-sample").
		WithDynakubeNamespaceSelector().
		ApiUrl(secretConfig.ApiUrl).
		ApplicationMonitoring(&dynatracev1beta1.ApplicationMonitoringSpec{
			UseCSIDriver: address.Of(false),
		})
	appDynakube := dynakubeBuilder.Build()

	namespaceBuilder := namespace.NewBuilder(sampleAppNamespace)

	sampleNamespace := namespaceBuilder.WithLabels(appOnlyDynakube.NamespaceSelector().MatchLabels).Build()
	sampleApp := sampleapps.NewSampleDeployment(t, appOnlyDynakube)
	sampleApp.WithNamespace(sampleNamespace)

	operatorNamespaceBuilder := namespace.NewBuilder(appOnlyDynakube.Namespace)

	assess.InstallOperatorFromSourceWithCustomNamespace(builder, operatorNamespaceBuilder.Build(), appOnlyDynakube)

	assess.InstallDynakubeWithTeardown(builder, &secretConfig, appOnlyDynakube)
	assess.InstallDynakubeWithTeardown(builder, &secretConfig, appDynakube)
	builder.Assess("install sample app", sampleApp.Install())

	builder.Assess("create additional pod without ownerreference and check injection", createAdditionalPodToCheckInjection())
	builder.Assess("create pod with already injected annotation and check injection", checkAlreadyInjected())
	builder.Assess("create pod with random user and check injection", checkInjectionWithRandomUser())

	builder.Teardown(sampleApp.UninstallNamespace())
	teardown.UninstallOperatorFromSource(builder, appOnlyDynakube)
	return builder.Feature()
}

func createAdditionalPodToCheckInjection() features.Func { //nolint:revive
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		resource := environmentConfig.Client().Resources()
		samplePod := manifests.ObjectFromFile[*corev1.Pod](t, path.Join(project.TestDataDir(), "sample-app/pod-base.yaml"))
		samplePod.Namespace = sampleAppNamespace
		require.NoError(t, resource.Create(ctx, samplePod))

		require.NoError(t, resource.Get(ctx, "php-sample", sampleAppNamespace, samplePod))

		require.NotNil(t, samplePod.Spec.InitContainers)

		for _, initContainer := range samplePod.Spec.InitContainers {
			require.Equal(t, "install-oneagent", initContainer.Name)
		}
		return ctx
	}
}

func checkAlreadyInjected() features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		resource := environmentConfig.Client().Resources()
		samplePod := manifests.ObjectFromFile[*corev1.Pod](t, path.Join(project.TestDataDir(), "sample-app/pod-base.yaml"))
		samplePod.Name = "php-sample-already-injected"
		samplePod.Namespace = sampleAppNamespace
		alreadyInjected := map[string]string{"oneagent.dynatrace.com/injected": "true"}
		samplePod.Annotations = alreadyInjected
		require.NoError(t, resource.Create(ctx, samplePod))

		require.NoError(t, resource.Get(ctx, "php-sample-already-injected", sampleAppNamespace, samplePod))

		require.Nil(t, samplePod.Spec.InitContainers)

		return ctx
	}
}

func checkInjectionWithRandomUser() features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		resource := environmentConfig.Client().Resources()
		samplePod := manifests.ObjectFromFile[*corev1.Pod](t, path.Join(project.TestDataDir(), "sample-app/pod-base.yaml"))
		samplePod.Name = "php-sample-random-user"
		samplePod.Namespace = sampleAppNamespace
		samplePod.Spec.SecurityContext.RunAsGroup = address.Of[int64](1234)
		samplePod.Spec.SecurityContext.RunAsUser = address.Of[int64](1234)

		require.NoError(t, resource.Create(ctx, samplePod))

		require.NoError(t, resource.Get(ctx, "php-sample-random-user", sampleAppNamespace, samplePod))

		require.NotNil(t, samplePod.Spec.InitContainers)

		for _, initContainer := range samplePod.Spec.InitContainers {
			require.Equal(t, "install-oneagent", initContainer.Name)
		}
		return ctx
	}
}

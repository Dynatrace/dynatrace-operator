//go:build e2e

package applicationmonitoring

import (
	"context"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/address"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sampleapps"
	sample "github.com/Dynatrace/dynatrace-operator/test/helpers/sampleapps/base"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps/assess"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps/teardown"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
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

	namespaceBuilder := namespace.NewBuilder(sampleAppNamespace)

	sampleNamespace := namespaceBuilder.WithLabels(appOnlyDynakube.NamespaceSelector().MatchLabels).Build()
	sampleApp := sampleapps.NewSampleDeployment(t, appOnlyDynakube)
	sampleApp.WithNamespace(sampleNamespace)

	operatorNamespaceBuilder := namespace.NewBuilder(appOnlyDynakube.Namespace)

	assess.InstallOperatorFromSourceWithCustomNamespace(builder, operatorNamespaceBuilder.Build(), appOnlyDynakube)

	assess.InstallDynakubeWithTeardown(builder, &secretConfig, appOnlyDynakube)
	builder.Assess("install sample app", sampleApp.Install())

	podSample := sampleapps.NewSampleDeployment(t, appOnlyDynakube)
	podSample.WithName("only-pod-sample")
	podSample.WithNamespace(sampleNamespace)
	builder.Assess("install additional pod", podSample.Install())
	builder.Assess("check injection of additional pod", checkInjection(podSample))

	alreadyInjectedSample := sampleapps.NewSampleDeployment(t, appOnlyDynakube)
	alreadyInjectedSample.WithName("already-injected")
	alreadyInjectedSample.WithNamespace(sampleNamespace)
	alreadyInjectedSample.WithAnnotations(map[string]string{"oneagent.dynatrace.com/injected": "true"})
	builder.Assess("install already injected sample app", alreadyInjectedSample.Install())
	builder.Assess("check if pods with already injection annotation are not injected again", checkAlreadyInjected(alreadyInjectedSample))

	randomUserSample := sampleapps.NewSampleDeployment(t, appOnlyDynakube)
	randomUserSample.WithName("random-user")
	randomUserSample.WithNamespace(sampleNamespace)
	randomUserSample.WithSecurityContext(corev1.PodSecurityContext{
		RunAsUser:  address.Of[int64](1234),
		RunAsGroup: address.Of[int64](1234),
	})
	builder.Assess("install sample app with random users set", randomUserSample.Install())
	builder.Assess("check injection of pods with random user", checkInjection(randomUserSample))

	builder.Teardown(sampleApp.UninstallNamespace())
	teardown.UninstallOperatorFromSource(builder, appOnlyDynakube)
	return builder.Feature()
}

func checkInjection(deployment sample.App) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		resource := environmentConfig.Client().Resources()
		samplePods := deployment.GetPods(ctx, t, resource)

		require.NotNil(t, samplePods)

		for _, item := range samplePods.Items {
			require.NotNil(t, item.Spec.InitContainers)
			require.Equal(t, "install-oneagent", item.Spec.InitContainers[0].Name)
		}
		return ctx
	}
}

func checkAlreadyInjected(deployment sample.App) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		resource := environmentConfig.Client().Resources()
		samplePods := deployment.GetPods(ctx, t, resource)

		require.NotNil(t, samplePods)

		for _, item := range samplePods.Items {
			require.Nil(t, item.Spec.InitContainers)
		}

		return ctx
	}
}

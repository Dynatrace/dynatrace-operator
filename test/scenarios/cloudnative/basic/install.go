//go:build e2e

package basic

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/istio"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sampleapps"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps/assess"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps/teardown"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/Dynatrace/dynatrace-operator/test/scenarios/cloudnative"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func Install(t *testing.T, istioEnabled bool) features.Feature {
	builder := features.New("default installation")
	t.Logf("istio enabled: %v", istioEnabled)
	secretConfig := tenant.GetSingleTenantSecret(t)

	dynakubeBuilder := dynakube.NewBuilder().
		WithDefaultObjectMeta().
		ApiUrl(secretConfig.ApiUrl).
		CloudNative(cloudnative.DefaultCloudNativeSpec())
	if istioEnabled {
		dynakubeBuilder = dynakubeBuilder.WithIstio()
	}
	testDynakube := dynakubeBuilder.Build()

	// Register operator install
	operatorNamespaceBuilder := namespace.NewBuilder(testDynakube.Namespace)
	if istioEnabled {
		operatorNamespaceBuilder = operatorNamespaceBuilder.WithLabels(istio.InjectionLabel)
	}
	assess.InstallOperatorFromSourceWithCustomNamespace(builder, operatorNamespaceBuilder.Build(), testDynakube)

	// Register sample app install
	namespaceBuilder := namespace.NewBuilder("cloudnative-sample")
	if istioEnabled {
		namespaceBuilder = namespaceBuilder.WithLabels(istio.InjectionLabel)
	}
	sampleNamespace := namespaceBuilder.Build()
	sampleApp := sampleapps.NewSampleDeployment(t, testDynakube)
	sampleApp.WithNamespace(sampleNamespace)
	builder.Assess("create sample namespace", sampleApp.InstallNamespace())

	// Register dynakube install
	assess.InstallDynakube(builder, &secretConfig, testDynakube)

	// Register sample app install
	builder.Assess("install sample app", sampleApp.Install())

	// Register actual test
	cloudnative.AssessSampleInitContainers(builder, sampleApp)
	if istioEnabled {
		istio.AssessIstio(builder, testDynakube, sampleApp)
	}

	// Register sample, dynakube and operator uninstall
	builder.Teardown(sampleApp.UninstallNamespace())
	teardown.UninstallDynatrace(builder, testDynakube)

	return builder.Feature()
}

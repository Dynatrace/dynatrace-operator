//go:build e2e

package _default

import (
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/codemodules"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/istio"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sampleapps"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/setup"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps/assess"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps/teardown"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/Dynatrace/dynatrace-operator/test/scenarios/cloudnative"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func Default(t *testing.T, istioEnabled bool) features.Feature {
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

	steps := setup.NewEnvironmentSetup(
		setup.CreateNamespaceWithoutTeardown(operatorNamespaceBuilder.Build()),
		setup.DeployOperatorViaMake(testDynakube.NeedsCSIDriver()))
	steps.CreateSetupSteps(builder)

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
	builder.Assess("install sample app", sampleApp.Install())

	// Register actual test
	cloudnative.AssessSampleInitContainers(builder, sampleApp)
	if istioEnabled {
		istio.AssessIstio(builder, testDynakube, sampleApp)
	}

	builder.Assess(fmt.Sprintf("check %s has no conn info", codemodules.RuxitAgentProcFile), codemodules.CheckRuxitAgentProcFileHasNoConnInfo(testDynakube))

	// Register sample, dynakube and operator uninstall
	builder.Teardown(sampleApp.UninstallNamespace())
	teardown.DeleteDynakube(builder, testDynakube)
	steps.CreateTeardownSteps(builder)

	return builder.Feature()
}

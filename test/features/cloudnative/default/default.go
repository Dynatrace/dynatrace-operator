//go:build e2e

package _default

import (
	"fmt"
	"github.com/Dynatrace/dynatrace-operator/test/features/cloudnative"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/codemodules"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/istio"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sample"

	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func Feature(t *testing.T, istioEnabled bool) features.Feature {
	builder := features.New("cloudnative default installation")
	// Register operator install
	if istioEnabled {
		builder.WithLabel("name", "cloudnative-istio")
	} else {
		builder.WithLabel("name", "cloudnative-default")
	}
	t.Logf("istio enabled: %v", istioEnabled)
	secretConfig := tenant.GetSingleTenantSecret(t)

	options := []dynakube.Option{
		dynakube.WithApiUrl(secretConfig.ApiUrl),
		dynakube.WithCloudNativeSpec(cloudnative.DefaultCloudNativeSpec()),
	}
	if istioEnabled {
		options = append(options, dynakube.WithIstioIntegration())
	}
	testDynakube := *dynakube.New(options...)
	// Register sample app install
	namespaceOptions := []namespace.Option{}
	if istioEnabled {
		namespaceOptions = append(namespaceOptions, namespace.WithIstio())
	}
	sampleNamespace := *namespace.New("cloudnative-sample", namespaceOptions...)
	sampleApp := sample.NewApp(t, &testDynakube, sample.WithNamespace(sampleNamespace), sample.AsDeployment())

	builder.Assess("create sample namespace", sampleApp.InstallNamespace())

	// Register dynakube install
	dynakube.Install(builder, helpers.LevelAssess, &secretConfig, testDynakube)
	builder.Assess("install sample app", sampleApp.Install())

	// Register actual test
	cloudnative.AssessSampleInitContainers(builder, sampleApp)
	if istioEnabled {
		istio.AssessIstio(builder, testDynakube, *sampleApp)
	}

	builder.Assess(fmt.Sprintf("check %s has no conn info", codemodules.RuxitAgentProcFile), codemodules.CheckRuxitAgentProcFileHasNoConnInfo(testDynakube))

	// Register sample, dynakube and operator uninstall
	builder.Teardown(sampleApp.Uninstall())
	dynakube.Delete(builder, helpers.LevelTeardown, testDynakube)
	return builder.Feature()
}

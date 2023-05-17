//go:build e2e

package assess

import (
	dynatracev1 "github.com/Dynatrace/dynatrace-operator/src/api/v1"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/manifests"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps/teardown"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func InstallManifest(builder *features.FeatureBuilder, deploymentPath string) {
	builder.Assess("installed manifests", manifests.InstallFromFile(deploymentPath))
}

func InstallDynatrace(builder *features.FeatureBuilder, secretConfig *tenant.Secret, testDynakube dynatracev1.DynaKube) {
	InstallOperatorFromSource(builder, testDynakube)
	InstallDynakube(builder, secretConfig, testDynakube)
}

func InstallDynatraceWithTeardown(builder *features.FeatureBuilder, secretConfig *tenant.Secret, testDynakube dynatracev1.DynaKube) {
	InstallDynatrace(builder, secretConfig, testDynakube)
	teardown.UninstallDynatrace(builder, testDynakube)
}

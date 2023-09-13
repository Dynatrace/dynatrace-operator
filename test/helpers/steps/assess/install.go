//go:build e2e

package assess

import (
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/manifests"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func InstallManifest(builder *features.FeatureBuilder, deploymentPath string) {
	builder.Assess("installed manifests", manifests.InstallFromFile(deploymentPath))
}

//
// func InstallDynatrace(builder *features.FeatureBuilder, secretConfig *tenant.Secret, testDynakube dynatracev1beta1.DynaKube) {
// 	InstallOperatorFromSource(builder, testDynakube)
// 	InstallDynakube(builder, secretConfig, testDynakube)
// }
//
// func InstallDynatraceWithTeardown(builder *features.FeatureBuilder, secretConfig *tenant.Secret, testDynakube dynatracev1beta1.DynaKube) {
// 	InstallDynatrace(builder, secretConfig, testDynakube)
// 	teardown.UninstallDynatrace(builder, testDynakube)
// }

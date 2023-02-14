//go:build e2e

package teardown

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/csi"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/operator"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func UninstallOperatorFromSource(builder *features.FeatureBuilder, testDynakube dynatracev1beta1.DynaKube) {
	addCsiCleanUp(builder, testDynakube)
	builder.WithTeardown("operator manifests uninstalled", operator.UninstallViaMake(testDynakube.NeedsCSIDriver()))
	builder.WithTeardown("deleted operator namespace", namespace.Delete(testDynakube.Namespace))
}

func UninstallOperatorFromRelease(builder *features.FeatureBuilder, testDynakube dynatracev1beta1.DynaKube, releaseTag string) {
	addCsiCleanUp(builder, testDynakube)
	builder.WithTeardown("operator manifests uninstalled", operator.UninstallFromGithub(releaseTag, testDynakube.NeedsCSIDriver()))
	builder.WithTeardown("deleted operator namespace", namespace.Delete(testDynakube.Namespace))
}

func addCsiCleanUp(builder *features.FeatureBuilder, testDynakube dynatracev1beta1.DynaKube) {
	if testDynakube.NeedsCSIDriver() {
		builder.WithTeardown("clean up csi driver files", csi.CleanUpEachPod(testDynakube.Namespace))
	}
}

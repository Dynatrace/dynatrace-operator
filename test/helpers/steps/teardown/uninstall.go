//go:build e2e

package teardown

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/manifests"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func UninstallManifest(builder *features.FeatureBuilder, deploymentPath string) {
	builder.WithTeardown("uninstalled manifests", manifests.UninstallFromFile(deploymentPath))
}

func UninstallDynatrace(builder *features.FeatureBuilder, testDynakube dynatracev1beta1.DynaKube) {
	DeleteDynakube(builder, testDynakube)
	UninstallOperator(builder, testDynakube)
}

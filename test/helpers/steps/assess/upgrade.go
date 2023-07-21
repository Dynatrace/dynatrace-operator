//go:build e2e

package assess

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/operator"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func UpgradeOperatorFromSource(builder *features.FeatureBuilder, testDynakube dynatracev1beta1.DynaKube) {
	builder.Assess("upgrading operator from source", operator.InstallViaMake(testDynakube.NeedsCSIDriver()))
	verifyOperatorDeployment(builder, testDynakube.Namespace, testDynakube.NeedsCSIDriver())
}

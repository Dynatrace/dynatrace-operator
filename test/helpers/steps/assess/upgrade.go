//go:build e2e

package assess

import (
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/operator"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func UpgradeOperatorFromSource(builder *features.FeatureBuilder, withCSIDriver bool) {
	builder.Assess("upgrading operator from source", operator.InstallViaMake(withCSIDriver))
	verifyOperatorDeployment(builder, withCSIDriver)
}

//go:build e2e

package assess

import (
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/csi"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/operator"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/webhook"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func VerifyOperatorDeployment(builder *features.FeatureBuilder, withCSIDriver bool) {
	builder.Assess("operator started", operator.WaitForDeployment(operator.DefaultNamespace))
	builder.Assess("webhook started", webhook.WaitForDeployment(operator.DefaultNamespace))
	if withCSIDriver {
		builder.Assess("csi driver started", csi.WaitForDaemonset(operator.DefaultNamespace))
	}
}

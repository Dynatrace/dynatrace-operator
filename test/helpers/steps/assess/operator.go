//go:build e2e

package assess

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/csi"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/operator"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/webhook"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func InstallOperatorFromSource(builder *features.FeatureBuilder, testDynakube dynatracev1beta1.DynaKube) {
	builder.Assess("create operator namespace", namespace.Create(namespace.NewBuilder(testDynakube.Namespace).Build()))
	builder.Assess("operator manifests installed", operator.InstallViaMake(testDynakube.NeedsCSIDriver()))
	verifyOperatorDeployment(builder, testDynakube)
}

func InstallOperatorFromRelease(builder *features.FeatureBuilder, testDynakube dynatracev1beta1.DynaKube, releaseTag string) {
	builder.Assess("create operator namespace", namespace.Create(namespace.NewBuilder(testDynakube.Namespace).Build()))
	builder.Assess("operator manifests installed", operator.InstallFromGithub(releaseTag, testDynakube.NeedsCSIDriver()))
	verifyOperatorDeployment(builder, testDynakube)
}

func verifyOperatorDeployment(builder *features.FeatureBuilder, testDynakube dynatracev1beta1.DynaKube) {
	builder.Assess("operator started", operator.WaitForDeployment(testDynakube.Namespace))
	builder.Assess("webhook started", webhook.WaitForDeployment(testDynakube.Namespace))
	if testDynakube.NeedsCSIDriver() {
		builder.Assess("csi driver started", csi.WaitForDaemonset(testDynakube.Namespace))
	}
}

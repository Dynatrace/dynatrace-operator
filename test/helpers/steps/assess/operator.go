//go:build e2e

package assess

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/csi"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/operator"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/webhook"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func InstallOperatorFromSource(builder *features.FeatureBuilder, testDynakube dynatracev1beta1.DynaKube) {
	namespaceBuilder := namespace.NewBuilder(testDynakube.Namespace)
	InstallOperatorFromSourceWithCustomNamespace(builder, namespaceBuilder.Build(), testDynakube)
}

func InstallOperatorFromSourceWithCustomNamespace(builder *features.FeatureBuilder, operatorNamespace corev1.Namespace, testDynakube dynatracev1beta1.DynaKube) {
	builder.Assess("create operator namespace", namespace.Create(operatorNamespace))
	builder.Assess("operator manifests installed", operator.InstallViaMake(testDynakube.NeedsCSIDriver()))
	verifyOperatorDeployment(builder, testDynakube)
}

func InstallOperatorFromRelease(builder *features.FeatureBuilder, testDynakube dynatracev1beta1.DynaKube, releaseTag string) {
	builder.Assess("create operator namespace", namespace.Create(namespace.NewBuilder(testDynakube.Namespace).Build()))
	builder.Assess("operator manifests installed", operator.InstallViaHelm(releaseTag, testDynakube.NeedsCSIDriver(), "dynatrace"))
	verifyOperatorDeployment(builder, testDynakube)
}

func verifyOperatorDeployment(builder *features.FeatureBuilder, testDynakube dynatracev1beta1.DynaKube) {
	builder.Assess("operator started", operator.WaitForDeployment(testDynakube.Namespace))
	builder.Assess("webhook started", webhook.WaitForDeployment(testDynakube.Namespace))
	if testDynakube.NeedsCSIDriver() {
		builder.Assess("csi driver started", csi.WaitForDaemonset(testDynakube.Namespace))
		builder.Assess("csi driver file system reset", csi.CleanUpEachPod(testDynakube.Namespace))
	}
}

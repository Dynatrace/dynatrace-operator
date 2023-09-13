//go:build e2e

package assess

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/csi"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/oneagent"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/operator"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/webhook"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

// func InstallOperatorFromSource(builder *features.FeatureBuilder, testDynakube dynatracev1beta1.DynaKube) {
// 	namespaceBuilder := namespace.NewBuilder(testDynakube.Namespace)
// 	InstallOperatorFromSourceWithCustomNamespace(builder, namespaceBuilder.Build(), testDynakube)
// }

// func InstallOperatorFromSourceWithCustomNamespace(builder *features.FeatureBuilder, operatorNamespace corev1.Namespace, testDynakube dynatracev1beta1.DynaKube) {
// 	builder.Assess("create operator namespace", namespace.Create(operatorNamespace))
// 	builder.Assess("operator manifests installed", operator.InstallViaMake(testDynakube.NeedsCSIDriver()))
// 	verifyOperatorDeployment(builder, testDynakube)
// }

// func InstallOperatorFromRelease(builder *features.FeatureBuilder, testDynakube dynatracev1beta1.DynaKube, releaseTag string) {
// 	builder.Assess("create operator namespace", namespace.Create(namespace.NewBuilder(testDynakube.Namespace).Build()))
// 	builder.Assess("operator manifests installed", operator.InstallViaHelm(releaseTag, testDynakube.NeedsCSIDriver(), "dynatrace"))
// 	verifyOperatorDeployment(builder, testDynakube)
// }

func AddClassicCleanUp(builder *features.FeatureBuilder, testDynakube dynatracev1beta1.DynaKube) {
	builder.Assess("clean up OneAgent files from nodes", oneagent.CreateUninstallDaemonSet(testDynakube))
	builder.Assess("wait for daemonset", oneagent.WaitForUninstallOneAgentDaemonset(testDynakube.Namespace))
	builder.Assess("OneAgent files removed from nodes", oneagent.CleanUpEachNode(testDynakube.Namespace))
}

func verifyOperatorDeployment(builder *features.FeatureBuilder, withCSIDriver bool) {
	builder.Assess("operator started", operator.WaitForDeployment(dynakube.DefaultNamespace))
	builder.Assess("webhook started", webhook.WaitForDeployment(dynakube.DefaultNamespace))
	if withCSIDriver {
		builder.Assess("csi driver started", csi.WaitForDaemonset(dynakube.DefaultNamespace))
	}
}

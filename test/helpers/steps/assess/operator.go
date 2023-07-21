//go:build e2e

package assess

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/csi"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/oneagent"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/operator"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/webhook"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func InstallOperatorFromSource(builder *features.FeatureBuilder, operatorNamespace string, useCsi bool) {
	namespaceBuilder := namespace.NewBuilder(operatorNamespace)
	InstallOperatorFromSourceWithCustomNamespace(builder, namespaceBuilder.Build(), useCsi)
}

func InstallOperatorFromSourceWithCustomNamespace(builder *features.FeatureBuilder, operatorNamespace corev1.Namespace, useCsi bool) {
	builder.Assess("create operator namespace", namespace.Create(operatorNamespace))
	builder.Assess("operator manifests installed", operator.InstallViaMake(useCsi))
	verifyOperatorDeployment(builder, operatorNamespace.Name, useCsi)
}

func InstallOperatorFromRelease(builder *features.FeatureBuilder, operatorNamespace string, useCsi bool, releaseTag string) {
	builder.Assess("create operator namespace", namespace.Create(namespace.NewBuilder(operatorNamespace).Build()))
	builder.Assess("operator manifests installed", operator.InstallViaHelm(releaseTag, useCsi, "dynatrace"))
	verifyOperatorDeployment(builder, operatorNamespace, useCsi)
}

func AddClassicCleanUp(builder *features.FeatureBuilder, testDynakube dynatracev1beta1.DynaKube) {
	builder.Assess("clean up OneAgent files from nodes", oneagent.CreateUninstallDaemonSet(testDynakube))
	builder.Assess("wait for daemonset", oneagent.WaitForUninstallOneAgentDaemonset(testDynakube.Namespace))
	builder.Assess("OneAgent files removed from nodes", oneagent.CleanUpEachNode(testDynakube.Namespace))
}

func verifyOperatorDeployment(builder *features.FeatureBuilder, namespace string, useCsi bool) {
	builder.Assess("operator started", operator.WaitForDeployment(namespace))
	builder.Assess("webhook started", webhook.WaitForDeployment(namespace))
	if useCsi {
		builder.Assess("csi driver started", csi.WaitForDaemonset(namespace))
	}
}

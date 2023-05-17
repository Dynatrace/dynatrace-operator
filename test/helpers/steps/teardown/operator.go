//go:build e2e

package teardown

import (
	dynatracev1 "github.com/Dynatrace/dynatrace-operator/src/api/v1"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/csi"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/oneagent"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/operator"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func UninstallOperatorFromSource(builder *features.FeatureBuilder, testDynakube dynatracev1.DynaKube) {
	addCsiCleanUp(builder, testDynakube)
	addNodeCleanUp(builder, testDynakube)
	builder.WithTeardown("operator manifests uninstalled", operator.UninstallViaMake(testDynakube.NeedsCSIDriver()))
	builder.WithTeardown("deleted operator namespace", namespace.Delete(testDynakube.Namespace))
}

func addCsiCleanUp(builder *features.FeatureBuilder, testDynakube dynatracev1.DynaKube) {
	if testDynakube.NeedsCSIDriver() {
		builder.WithTeardown("clean up csi driver files", csi.CleanUpEachPod(testDynakube.Namespace))
	}
}

func addNodeCleanUp(builder *features.FeatureBuilder, testDynakube dynatracev1.DynaKube) {
	if testDynakube.ClassicFullStackMode() {
		builder.WithTeardown("clean up OneAgent files from nodes", oneagent.CreateUninstallDaemonSet(testDynakube))
		builder.WithTeardown("wait for daemonset", oneagent.WaitForUninstallOneAgentDaemonset(testDynakube.Namespace))
		builder.WithTeardown("OneAgent files removed from nodes", oneagent.CleanUpEachNode(testDynakube.Namespace))
	}
}

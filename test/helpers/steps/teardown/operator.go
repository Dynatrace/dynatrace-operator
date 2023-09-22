//go:build e2e

package teardown

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/oneagent"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/operator"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func AddClassicCleanUp(builder *features.FeatureBuilder, testDynakube dynatracev1beta1.DynaKube) {
	builder.WithTeardown("clean up OneAgent files from nodes", oneagent.CreateUninstallDaemonSet(testDynakube))
	builder.WithTeardown("wait for daemonset", oneagent.WaitForUninstallOneAgentDaemonset(testDynakube.Namespace))
	builder.WithTeardown("OneAgent files removed from nodes", oneagent.CleanUpEachNode(testDynakube.Namespace))
}

func UninstallOperatorWithEdgeConnectFromSource(builder *features.FeatureBuilder, useCsi bool) {
	builder.WithTeardown("operator manifests uninstalled", operator.UninstallViaMake(useCsi))
}

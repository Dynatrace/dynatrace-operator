//go:build e2e

package teardown

import (
	edgeconnectv1alpha1 "github.com/Dynatrace/dynatrace-operator/src/api/v1alpha1/edgeconnect"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/oneagent"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/operator"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func DeleteDynakube(builder *features.FeatureBuilder, testDynakube dynatracev1beta1.DynaKube) {
	builder.WithTeardown("dynakube deleted", dynakube.Delete(testDynakube))
	if testDynakube.NeedsOneAgent() {
		builder.WithTeardown("oneagent pods stopped", oneagent.WaitForDaemonSetPodsDeletion(testDynakube))
	}
}

func DeleteEdgeConnect(builder *features.FeatureBuilder, testEdgeConnect edgeconnectv1alpha1.EdgeConnect) {
	builder.WithTeardown("edgeconnect deleted", edgeconnect.Delete(testEdgeConnect))
	builder.WithTeardown("operator manifests uninstalled", operator.UninstallViaMake(false))
}

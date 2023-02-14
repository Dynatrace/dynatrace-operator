//go:build e2e

package teardown

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/oneagent"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func DeleteDynakube(builder *features.FeatureBuilder, testDynakube dynatracev1beta1.DynaKube) {
	builder.WithTeardown("dynakube deleted", dynakube.Delete(testDynakube))
	if testDynakube.NeedsOneAgent() {
		builder.WithTeardown("oneagent pods stopped", oneagent.WaitForDaemonSetPodsDeletion(testDynakube))
	}
}

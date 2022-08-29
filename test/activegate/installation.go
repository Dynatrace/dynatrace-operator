package activegate

import (
	"github.com/Dynatrace/dynatrace-operator/test/statefulset"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func WaitForStatefulSet() features.Func {
	return statefulset.WaitFor("dynakube-activegate", "dynatrace")
}

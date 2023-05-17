package validation

import (
	dynatracev1 "github.com/Dynatrace/dynatrace-operator/src/api/v1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/istio"
)

const (
	errorNoResources = `No resources for istio available`
)

func noResourcesAvailable(dv *dynakubeValidator, dynakube *dynatracev1.DynaKube) string {
	if dynakube.Spec.EnableIstio {
		enabled, err := istio.CheckIstioEnabled(dv.cfg)
		if !enabled || err != nil {
			return errorNoResources
		}
	}

	return ""
}

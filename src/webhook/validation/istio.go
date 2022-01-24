package validation

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
)

const (
	errorNoResources       = `No resources for istio available`
	errorIstioNotAvailable = `Istio CRD is not available`
)

func noResourcesAvailable(dv *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	if dynakube.Spec.EnableIstio {
		_, err := kubeobjects.CheckIstioEnabled(dv.cfg)
		if err != nil {
			return errorNoResources
		}
	}

	return ""
}

func istioCRDNotAvailable(dv *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	if dynakube.Spec.EnableIstio {
		probe := kubeobjects.VerifyIstioCrdAvailability(dynakube, dv.cfg)
		if probe == kubeobjects.ProbeTypeNotFound {
			return errorIstioNotAvailable
		}
	}

	return ""
}

package validation

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"k8s.io/client-go/rest"
)

const errorNoResources = `No resources for istio available`

func noResourcesAvailable(dv *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	cfg := &rest.Config{}
	if dynakube.Spec.EnableIstio {
		_, err := kubeobjects.CheckIstioEnabled(cfg)
		if err != nil {
			return errorNoResources
		}

		probe := kubeobjects.VerifyIstioCrdAvailability(dynakube, cfg)
		if probe == kubeobjects.ProbeTypeNotFound {
			return errorNoResources
		}
	} else {
		return errorNoResources
	}

	return ""
}

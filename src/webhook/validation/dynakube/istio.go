package dynakube

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/istio"
	istioclientset "istio.io/client-go/pkg/clientset/versioned"
)

const (
	errorNoResources           = `No resources for istio available`
	errorFailToInitIstioClient = `Failed to initialize istio client`
)

func noResourcesAvailable(dv *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	if dynakube.Spec.EnableIstio {
		ic, err := istioclientset.NewForConfig(dv.cfg)

		if err != nil {
			return errorFailToInitIstioClient
		}
		enabled, err := istio.CheckIstioInstalled(ic.Discovery())
		if !enabled || err != nil {
			return errorNoResources
		}
	}

	return ""
}

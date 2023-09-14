package dynakube

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/istio"
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
)

const (
	errorNoResources           = `No resources for istio available`
	errorFailToInitIstioClient = `Failed to initialize istio client`
)

func noResourcesAvailable(dv *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	if dynakube.Spec.EnableIstio {
		istioClient, err := istio.NewClient(dv.cfg, scheme.Scheme, dynakube.Namespace)
		if err != nil {
			return errorFailToInitIstioClient
		}
		enabled, err := istioClient.CheckIstioInstalled()
		if !enabled || err != nil {
			return errorNoResources
		}
	}

	return ""
}

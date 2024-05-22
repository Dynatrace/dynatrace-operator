package dynakube

import (
	"context"

	dynatracev1beta2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/istio"
)

const (
	errorNoResources           = `No resources for istio available`
	errorFailToInitIstioClient = `Failed to initialize istio client`
)

func noResourcesAvailable(_ context.Context, dv *dynakubeValidator, dynakube *dynatracev1beta2.DynaKube) string {
	if dynakube.Spec.EnableIstio {
		istioClient, err := istio.NewClient(dv.cfg, dynakube)
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

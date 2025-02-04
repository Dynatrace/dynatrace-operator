package validation

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/istio"
)

const (
	errorNoResources           = `No resources for istio available`
	errorFailToInitIstioClient = `Failed to initialize istio client`
)

func noResourcesAvailable(_ context.Context, dv *Validator, dk *dynakube.DynaKube) string {
	if dk.Spec.EnableIstio {
		istioClient, err := istio.NewClient(dv.cfg, dk)
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

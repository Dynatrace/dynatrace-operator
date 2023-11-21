package dynakube

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/istio"
)

const (
	errorNoResources           = `No resources for istio available`
	errorFailToInitIstioClient = `Failed to initialize istio client`
)

func noResourcesAvailable(_ context.Context, dv *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	if dynakube.Spec.EnableIstio {
		istioClient, err := istio.NewClient(dv.cfg, scheme.Scheme, dynakube)
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

package validation

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/istio"
)

const (
	errorNoIstioInstalled      = `No resources for istio available`
	errorFailedToCheckForIstio = `Failed to verify if istio is available`
)

func noIstioInstalled(_ context.Context, dv *Validator, dk *dynakube.DynaKube) string {
	if dk.Spec.EnableIstio {
		enabled, err := istio.IsInstalled(dv.discoveryClient)
		if err != nil {
			return errorFailedToCheckForIstio
		}

		if !enabled {
			return errorNoIstioInstalled
		}
	}

	return ""
}

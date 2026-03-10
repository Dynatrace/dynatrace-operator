package validation

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/istio"
)

const (
	errorNoIstioInstalled = `No resources for istio available`
)

func isIstioNotInstalled(ctx context.Context, dv *Validator, dk *dynakube.DynaKube) string {
	if dk.Spec.EnableIstio && !istio.IsInstalled(ctx, dv.apiReader) {
		return errorNoIstioInstalled
	}

	return ""
}

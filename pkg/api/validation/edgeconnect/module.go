package validation

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
)

const (
	errorModuleDisabled = `EdgeConnect has been disabled during Operator install. The necessary resources for EdgeConnect to work are not present on the cluster. Redeploy the Operator with all the necessary resources`
)

func isModuleDisabled(_ context.Context, v *Validator, _ *edgeconnect.EdgeConnect) string {
	if v.modules.EdgeConnect {
		return ""
	}

	return errorModuleDisabled
}

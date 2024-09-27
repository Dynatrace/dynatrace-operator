package validation

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/installconfig"
)

var (
	errorModuleDisabled = installconfig.GetModuleValidationErrorMessage("EdgeConnect")
)

func isModuleDisabled(_ context.Context, v *Validator, _ *edgeconnect.EdgeConnect) string {
	if v.modules.EdgeConnect {
		return ""
	}

	return errorModuleDisabled
}

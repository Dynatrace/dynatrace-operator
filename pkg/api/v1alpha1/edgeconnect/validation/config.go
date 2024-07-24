package validation

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

const (
	testServiceAccountName = "test"
)

var log = logd.Get().WithName("edgeconnect-validation")

type validator func(ctx context.Context, dv *edgeconnectValidator, ec *edgeconnect.EdgeConnect) string

var validators = []validator{
	isInvalidApiServer,
	nameTooLong,
	checkHostPatternsValue,
	isInvalidServiceName,
	automationRequiresProvisionerValidation,
}

package edgeconnect

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/logger"
)

var log = logger.Get().WithName("edgeconnect-validation")

type validator func(ctx context.Context, dv *edgeconnectValidator, edgeConnect *edgeconnect.EdgeConnect) string

var validators = []validator{
	isInvalidApiServer,
	nameTooLong,
	checkHostPatternsValue,
}

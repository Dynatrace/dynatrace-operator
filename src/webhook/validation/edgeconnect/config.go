package edgeconnect

import (
	"github.com/Dynatrace/dynatrace-operator/src/api/v1alpha1/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/src/logger"
)

var log = logger.Factory.GetLogger("edgeconnect-validation")

type validator func(dv *edgeconnectValidator, edgeConnect *edgeconnect.EdgeConnect) string

var validators = []validator{
	IsInvalidApiServer,
}

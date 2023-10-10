package edgeconnect

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/logger"
)

var log = logger.Factory.GetLogger("edgeconnect-validation")

type validator func(dv *edgeconnectValidator, edgeConnect *edgeconnect.EdgeConnect) string

var validators = []validator{
	IsInvalidApiServer,
}

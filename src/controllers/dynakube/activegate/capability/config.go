package capability

import (
	"github.com/Dynatrace/dynatrace-operator/src/logger"
)

const (
	HttpsServicePortName = "https"
	HttpsServicePort     = 443
	HttpServicePortName  = "http"
	HttpServicePort      = 80
)

var log = logger.NewDTLogger().WithName("dynakube-activegate-capability")

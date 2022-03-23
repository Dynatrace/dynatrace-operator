package initgeneration

import (
	"github.com/Dynatrace/dynatrace-operator/src/logger"
)

var (
	log = logger.NewDTLogger().WithName("initgeneration")
)

const (
	clusterCaKey    = "certs"
	proxyKey        = "proxy"
	activeGateCaKey = "server.crt"
)

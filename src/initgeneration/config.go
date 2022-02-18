package initgeneration

import (
	"github.com/Dynatrace/dynatrace-operator/src/logger"
)

var (
	log = logger.NewDTLogger().WithName("initgeneration")
)

const (
	trustedCAKey = "certs"
	proxyKey     = "proxy"
	tlsCertKey   = "server.crt"
)

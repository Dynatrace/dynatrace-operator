package initgeneration

import (
	"github.com/Dynatrace/dynatrace-operator/src/logger"
)

var (
	log = logger.NewDTLogger().WithName("initgeneration")
)

const (
	trustedCASecretField = "certs"
	proxyInitSecretField = "proxy"
	tlsCertKey           = "server.crt"
)

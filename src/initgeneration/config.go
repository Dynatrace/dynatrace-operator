package initgeneration

import (
	_ "embed"
	"text/template"

	"github.com/Dynatrace/dynatrace-operator/src/logger"
)

var (
	log = logger.NewDTLogger().WithName("initgeneration")

	//go:embed init.sh.tmpl
	scriptContent string
	scriptTmpl    = template.Must(template.New("initScript").Parse(scriptContent))
)

const (
	notMappedIM              = "-"
	trustedCASecretField     = "certs"
	proxyInitSecretField     = "proxy"
	trustedCAInitSecretField = "ca.pem"
	initScriptSecretField    = "init.sh"
	tlsCertKey               = "server.crt"
)

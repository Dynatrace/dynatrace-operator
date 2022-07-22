package troubleshoot

import (
	"net/http"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type troubleshootContext struct {
	apiReader              client.Reader
	httpClient             *http.Client
	namespaceName          string // the default namespace ("dynatrace") or provided in the command line
	dynakubeName           string // the default name of dynakube ("dynakube") or provided in the command line
	dynatraceApiSecretName string // the default name of dynatrace api secret or custom name
	customPullSecretName   string // the default name of pull secret ("dynakube-pull-secret") or custom name
	proxySecretName        string // name of proxy-secret
}

type troubleshootFunc func(troubleshootCtx *troubleshootContext) error

var (
	tslog = NewLogger("")
)

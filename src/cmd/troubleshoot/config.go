package troubleshoot

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type TestData struct {
	namespaceName          string // the default namespace ("dynatrace") or provided in the command line
	dynakubeName           string // the default name of dynakube ("dynakube") or provided in the command line
	dynatraceApiSecretName string // the default name of dynatrace api secret or custom name
	customPullSecretName   string // the default name of pull secret ("dynakube-pull-secret") or custom name
	proxySecretName        string // name of proxy-secret
}

type TestFunc func(apiReader client.Reader, troubleshootContext *TestData) error

var (
	tslog = NewLogger("")
)

package troubleshoot

import (
	"net/http"

	"github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type troubleshootContext struct {
	apiReader              client.Reader
	httpClient             *http.Client
	namespaceName          string // the default namespace ("dynatrace") or provided in the command line
	dynakubeName           string // the default name of dynakube ("dynakube") or provided in the command line
	dynatraceApiSecretName string // the default name of dynatrace api secret or custom name
	pullSecretName         string // the default name of pull secret ("dynakube-pull-secret") or custom name
	proxySecretName        string // name of proxy-secret
	dynakube               v1beta1.DynaKube
	dynatraceApiSecret     corev1.Secret
	pullSecret             corev1.Secret
	proxySecret            corev1.Secret
}

type troubleshootFunc func(troubleshootCtx *troubleshootContext) error

var (
	log = newTroubleshootLogger("[          ]")
)

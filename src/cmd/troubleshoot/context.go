package troubleshoot

import (
	"github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"net/http"
	"net/url"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type troubleshootContext struct {
	apiReader          client.Reader
	httpClient         *http.Client
	namespaceName      string // the default namespace ("dynatrace") or provided in the command line
	dynakubeName       string // the default name of dynakube ("dynakube") or provided in the command line
	pullSecretName     string // the default name of pull secret ("dynakube-pull-secret") or custom name
	proxySecretName    string // name of proxy-secret
	dynakube           v1beta1.DynaKube
	dynatraceApiSecret corev1.Secret
	pullSecret         corev1.Secret
	proxySecret        corev1.Secret
	proxy              string
}

type troubleshootFunc func(troubleshootCtx *troubleshootContext) error

func (troubleshootCtx *troubleshootContext) SetTransportProxy(proxy string) error {
	if proxy != "" {
		proxyUrl, err := url.Parse(proxy)
		if err != nil {
			return errorWithMessagef(err, "could not parse proxy URL!")
		}

		troubleshootCtx.httpClient.Transport.(*http.Transport).Proxy = http.ProxyURL(proxyUrl)
		logInfof("using  '%s' proxy to connect to the registry", proxyUrl.Host)
	}

	return nil
}

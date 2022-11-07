package troubleshoot

import (
	"context"
	"net/http"
	"net/url"

	"github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type troubleshootContext struct {
	context            context.Context
	apiReader          client.Reader
	httpClient         *http.Client
	namespaceName      string // the default namespace ("dynatrace") or value provided in the command line
	dynakubeName       string // all dynakubes in namespaceName or value provided in the command line
	dynakube           v1beta1.DynaKube
	dynatraceApiSecret corev1.Secret
	pullSecret         corev1.Secret
	proxySecret        corev1.Secret
}

type troubleshootFunc func(troubleshootCtx *troubleshootContext) error

func (troubleshootCtx *troubleshootContext) SetTransportProxy(proxy string) error {
	if proxy != "" {
		proxyUrl, err := url.Parse(proxy)
		if err != nil {
			return errorWithMessagef(err, "could not parse proxy URL!")
		}

		if troubleshootCtx.httpClient.Transport == nil {
			troubleshootCtx.httpClient.Transport = http.DefaultTransport
		}

		troubleshootCtx.httpClient.Transport.(*http.Transport).Proxy = http.ProxyURL(proxyUrl)
		logInfof("using '%s' proxy to connect to the registry", proxyUrl.Host)
	}

	return nil
}

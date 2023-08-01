package troubleshoot

import (
	"context"
	"net/http"
	"net/url"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/token"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type troubleshootContext struct {
	context                  context.Context
	apiReader                client.Reader
	httpClient               *http.Client
	namespaceName            string // the default namespace ("dynatrace") or provided in the command line
	dynakube                 dynatracev1beta1.DynaKube
	dynatraceApiSecretTokens token.Tokens
	pullSecret               corev1.Secret
	kubeConfig               rest.Config
	fs                       afero.Afero
	baseLog                  logr.Logger
}

func (troubleshootCtx *troubleshootContext) SetTransportProxy(log logr.Logger, proxy string) error {
	if proxy != "" {
		proxyUrl, err := url.Parse(proxy)
		if err != nil {
			return errors.Wrap(err, "could not parse proxy URL!")
		}

		if troubleshootCtx.httpClient.Transport == nil {
			troubleshootCtx.httpClient.Transport = http.DefaultTransport
		}

		troubleshootCtx.httpClient.Transport.(*http.Transport).Proxy = http.ProxyURL(proxyUrl)
		logInfof(log, "using '%s' proxy to connect to the registry", proxyUrl.Host)
	}

	return nil
}

package dynatraceclient

import (
	"context"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type options struct {
	ctx  context.Context
	Opts []dtclient.Option
}

type Properties struct {
	ctx                 context.Context
	ApiReader           client.Reader
	Tokens              token.Tokens
	Proxy               *proxy
	ApiUrl              string
	Namespace           string
	NetworkZone         string
	TrustedCerts        string
	SkipCertCheck       bool
	DisableHostRequests bool
}

type proxy struct {
	Value     string
	ValueFrom string
}

func NewProperties(ctx context.Context, apiReader client.Reader, dynakube dynatracev1beta1.DynaKube, tokens token.Tokens) Properties {
	return Properties{
		ctx:                 ctx,
		ApiReader:           apiReader,
		Tokens:              tokens,
		ApiUrl:              dynakube.Spec.APIURL,
		Namespace:           dynakube.Namespace,
		Proxy:               convertProxy(dynakube.Spec.Proxy),
		NetworkZone:         dynakube.Spec.NetworkZone,
		TrustedCerts:        dynakube.Spec.TrustedCAs,
		SkipCertCheck:       dynakube.Spec.SkipCertCheck,
		DisableHostRequests: dynakube.FeatureDisableHostsRequests(),
	}
}

func convertProxy(dynakubeProxy *dynatracev1beta1.DynaKubeProxy) *proxy {
	if dynakubeProxy == nil {
		return nil
	}
	return &proxy{
		Value:     dynakubeProxy.Value,
		ValueFrom: dynakubeProxy.ValueFrom,
	}
}

// BuildDynatraceClient creates a new Dynatrace client using the settings configured on the given instance.
func BuildDynatraceClient(properties Properties) (dtclient.Client, error) {
	namespace := properties.Namespace
	apiReader := properties.ApiReader

	opts := newOptions(properties.ctx)
	opts.appendCertCheck(properties.SkipCertCheck)
	opts.appendNetworkZone(properties.NetworkZone)
	opts.appendDisableHostsRequests(properties.DisableHostRequests)

	err := opts.appendProxySettings(apiReader, properties.Proxy, namespace)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	err = opts.appendTrustedCerts(apiReader, properties.TrustedCerts, namespace)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return dtclient.NewClient(properties.ApiUrl, properties.Tokens.ApiToken().Value, properties.Tokens.PaasToken().Value, opts.Opts...)
}

func newOptions(ctx context.Context) *options {
	return &options{
		Opts: []dtclient.Option{},
		ctx:  ctx,
	}
}

// StaticDynatraceClient creates a dynatraceClientFunc always returning c.
func StaticDynatraceClient(c dtclient.Client) BuildFunc {
	return func(properties Properties) (dtclient.Client, error) {
		return c, nil
	}
}

func (opts *options) appendNetworkZone(networkZone string) {
	if networkZone != "" {
		opts.Opts = append(opts.Opts, dtclient.NetworkZone(networkZone))
	}
}

func (opts *options) appendCertCheck(skipCertCheck bool) {
	opts.Opts = append(opts.Opts, dtclient.SkipCertificateValidation(skipCertCheck))
}

func (opts *options) appendDisableHostsRequests(disableHostsRequests bool) {
	opts.Opts = append(opts.Opts, dtclient.DisableHostsRequests(disableHostsRequests))
}

func (opts *options) appendProxySettings(apiReader client.Reader, proxyEntry *proxy, namespace string) error {
	if p := proxyEntry; p != nil {
		if p.ValueFrom != "" {
			proxySecret := &corev1.Secret{}
			err := apiReader.Get(opts.ctx, client.ObjectKey{Name: p.ValueFrom, Namespace: namespace}, proxySecret)
			if err != nil {
				return errors.WithMessage(err, "failed to get proxy secret")
			}

			proxyURL, err := kubeobjects.ExtractToken(proxySecret, dtclient.CustomProxySecretKey)
			if err != nil {
				return errors.WithMessage(err, "failed to extract proxy secret field")
			}
			opts.Opts = append(opts.Opts, dtclient.Proxy(proxyURL))
		} else if p.Value != "" {
			opts.Opts = append(opts.Opts, dtclient.Proxy(p.Value))
		}
	}
	return nil
}

func (opts *options) appendTrustedCerts(apiReader client.Reader, trustedCerts string, namespace string) error {
	if trustedCerts != "" {
		certs := &corev1.ConfigMap{}
		if err := apiReader.Get(opts.ctx, client.ObjectKey{Namespace: namespace, Name: trustedCerts}, certs); err != nil {
			return errors.WithMessage(err, "failed to get certificate configmap")
		}
		if certs.Data[dtclient.CustomCertificatesConfigMapKey] == "" {
			return errors.New("failed to extract certificate configmap field: missing field certs")
		}
		opts.Opts = append(opts.Opts, dtclient.Certs([]byte(certs.Data[dtclient.CustomCertificatesConfigMapKey])))
	}
	return nil
}

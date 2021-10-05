package dynakube

import (
	"context"
	"fmt"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/controllers/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type options struct {
	Opts []dtclient.Option
}

type DynatraceClientProperties struct {
	ApiReader           client.Reader
	Secret              *corev1.Secret
	Proxy               *DynatraceClientProxy
	ApiUrl              string
	Namespace           string
	NetworkZone         string
	TrustedCerts        string
	SkipCertCheck       bool
	DisableHostRequests bool
}

type DynatraceClientProxy struct {
	Value     string
	ValueFrom string
}

const (
	proxy        = "proxy"
	certificates = "certs"
)

func NewDynatraceClientProperties(ctx context.Context, apiReader client.Reader, dk dynatracev1beta1.DynaKube) (*DynatraceClientProperties, error) {
	var tokens corev1.Secret
	var err error
	if err = apiReader.Get(ctx, client.ObjectKey{Name: dk.Tokens(), Namespace: dk.Namespace}, &tokens); err != nil {
		err = fmt.Errorf("failed to query tokens: %w", err)
	}
	return &DynatraceClientProperties{
		ApiReader:           apiReader,
		Secret:              &tokens,
		ApiUrl:              dk.Spec.APIURL,
		Namespace:           dk.Namespace,
		Proxy:               (*DynatraceClientProxy)(dk.Spec.Proxy),
		NetworkZone:         dk.Spec.NetworkZone,
		TrustedCerts:        dk.Spec.TrustedCAs,
		SkipCertCheck:       dk.Spec.SkipCertCheck,
		DisableHostRequests: dk.FeatureDisableHostsRequests(),
	}, err
}

// BuildDynatraceClient creates a new Dynatrace client using the settings configured on the given instance.
func BuildDynatraceClient(properties DynatraceClientProperties) (dtclient.Client, error) {
	namespace := properties.Namespace
	secret := properties.Secret
	apiReader := properties.ApiReader

	tokens, err := kubeobjects.NewTokens(secret)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	opts := newOptions()
	opts.appendCertCheck(properties.SkipCertCheck)
	opts.appendNetworkZone(properties.NetworkZone)
	opts.appendDisableHostsRequests(properties.DisableHostRequests)

	err = opts.appendProxySettings(apiReader, properties.Proxy, namespace)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	err = opts.appendTrustedCerts(apiReader, properties.TrustedCerts, namespace)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return dtclient.NewClient(properties.ApiUrl, tokens.ApiToken, tokens.PaasToken, opts.Opts...)
}

func newOptions() *options {
	return &options{
		Opts: []dtclient.Option{},
	}
}

// StaticDynatraceClient creates a DynatraceClientFunc always returning c.
func StaticDynatraceClient(c dtclient.Client) DynatraceClientFunc {
	return func(properties DynatraceClientProperties) (dtclient.Client, error) {
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

func (opts *options) appendProxySettings(apiReader client.Reader, proxyEntry *DynatraceClientProxy, namespace string) error {
	if p := proxyEntry; p != nil {
		if p.ValueFrom != "" {
			proxySecret := &corev1.Secret{}
			err := apiReader.Get(context.TODO(), client.ObjectKey{Name: p.ValueFrom, Namespace: namespace}, proxySecret)
			if err != nil {
				return fmt.Errorf("failed to get proxy secret: %w", err)
			}

			proxyURL, err := kubeobjects.ExtractToken(proxySecret, proxy)
			if err != nil {
				return fmt.Errorf("failed to extract proxy secret field: %w", err)
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
		if err := apiReader.Get(context.TODO(), client.ObjectKey{Namespace: namespace, Name: trustedCerts}, certs); err != nil {
			return fmt.Errorf("failed to get certificate configmap: %w", err)
		}
		if certs.Data[certificates] == "" {
			return fmt.Errorf("failed to extract certificate configmap field: missing field certs")
		}
		opts.Opts = append(opts.Opts, dtclient.Certs([]byte(certs.Data[certificates])))
	}
	return nil
}

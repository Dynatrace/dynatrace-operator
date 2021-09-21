package dynakube

import (
	"context"
	"fmt"

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
	Client              client.Client
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

// BuildDynatraceClient creates a new Dynatrace client using the settings configured on the given instance.
func BuildDynatraceClient(properties DynatraceClientProperties) (dtclient.Client, error) {
	namespace := properties.Namespace
	secret := properties.Secret
	clt := properties.Client

	tokens, err := kubeobjects.NewTokens(secret)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	opts := newOptions()
	opts.appendCertCheck(properties.SkipCertCheck)
	opts.appendNetworkZone(properties.NetworkZone)
	opts.appendDisableHostsRequests(properties.DisableHostRequests)

	err = opts.appendProxySettings(clt, properties.Proxy, namespace)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	err = opts.appendTrustedCerts(clt, properties.TrustedCerts, namespace)
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

func (opts *options) appendProxySettings(rtc client.Client, proxy *DynatraceClientProxy, namespace string) error {
	if p := proxy; p != nil {
		if p.ValueFrom != "" {
			proxySecret := &corev1.Secret{}
			err := rtc.Get(context.TODO(), client.ObjectKey{Name: p.ValueFrom, Namespace: namespace}, proxySecret)
			if err != nil {
				return fmt.Errorf("failed to get proxy secret: %w", err)
			}

			proxyURL, err := kubeobjects.ExtractToken(proxySecret, Proxy)
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

func (opts *options) appendTrustedCerts(rtc client.Client, trustedCerts string, namespace string) error {
	if trustedCerts != "" {
		certs := &corev1.ConfigMap{}
		if err := rtc.Get(context.TODO(), client.ObjectKey{Namespace: namespace, Name: trustedCerts}, certs); err != nil {
			return fmt.Errorf("failed to get certificate configmap: %w", err)
		}
		if certs.Data[Certificates] == "" {
			return fmt.Errorf("failed to extract certificate configmap field: missing field certs")
		}
		opts.Opts = append(opts.Opts, dtclient.Certs([]byte(certs.Data[Certificates])))
	}
	return nil
}

const (
	Proxy        = "proxy"
	Certificates = "certs"
)

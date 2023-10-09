package dynatraceclient

import (
	"context"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type options struct {
	ctx  context.Context
	Opts []dynatrace.Option
}

func newOptions(ctx context.Context) *options {
	return &options{
		Opts: []dynatrace.Option{},
		ctx:  ctx,
	}
}

func (opts *options) appendNetworkZone(networkZone string) {
	if networkZone != "" {
		opts.Opts = append(opts.Opts, dynatrace.NetworkZone(networkZone))
	}
}

func (opts *options) appendCertCheck(skipCertCheck bool) {
	opts.Opts = append(opts.Opts, dynatrace.SkipCertificateValidation(skipCertCheck))
}

func (opts *options) appendDisableHostsRequests(disableHostsRequests bool) {
	opts.Opts = append(opts.Opts, dynatrace.DisableHostsRequests(disableHostsRequests))
}

func (opts *options) appendProxySettings(apiReader client.Reader, dynakube *dynatracev1beta1.DynaKube) error {
	if dynakube == nil || !dynakube.HasProxy() {
		return nil
	}

	proxyOption, err := opts.createProxyOption(apiReader, dynakube)
	if err != nil {
		return err
	}

	opts.Opts = append(opts.Opts, proxyOption)
	return nil
}

func (opts *options) createProxyOption(apiReader client.Reader, dynakube *dynatracev1beta1.DynaKube) (dynatrace.Option, error) {
	var proxyOption dynatrace.Option

	proxyUrl, err := dynakube.Proxy(opts.ctx, apiReader)
	if err != nil {
		return proxyOption, err
	}

	proxyOption = dynatrace.Proxy(proxyUrl, dynakube.FeatureNoProxy())
	return proxyOption, nil
}

func (opts *options) appendTrustedCerts(apiReader client.Reader, trustedCerts string, namespace string) error {
	if trustedCerts != "" {
		certs := &corev1.ConfigMap{}
		if err := apiReader.Get(opts.ctx, client.ObjectKey{Namespace: namespace, Name: trustedCerts}, certs); err != nil {
			return errors.WithMessage(err, "failed to get certificate configmap")
		}
		if certs.Data[dynatracev1beta1.TrustedCAKey] == "" {
			return errors.New("failed to extract certificate configmap field: missing field certs")
		}
		opts.Opts = append(opts.Opts, dynatrace.Certs([]byte(certs.Data[dynatracev1beta1.TrustedCAKey])))
	}
	return nil
}

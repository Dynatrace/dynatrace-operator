package dynatraceclient //nolint:dupl

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
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
		opts.Opts = append(opts.Opts, dynatrace.WithNetworkZone(networkZone))
	}
}

func (opts *options) appendHostGroup(hostGroup string) {
	if hostGroup != "" {
		opts.Opts = append(opts.Opts, dynatrace.WithHostGroup(hostGroup))
	}
}

func (opts *options) appendCertCheck(skipCertCheck bool) {
	opts.Opts = append(opts.Opts, dynatrace.WithSkipCertificateValidation(skipCertCheck))
}

func (opts *options) appendProxySettings(apiReader client.Reader, dk *dynakube.DynaKube) error {
	if dk == nil || !dk.HasProxy() {
		return nil
	}

	proxyOption, err := opts.createProxyOption(apiReader, dk)
	if err != nil {
		return err
	}

	opts.Opts = append(opts.Opts, proxyOption)

	return nil
}

func (opts *options) createProxyOption(apiReader client.Reader, dk *dynakube.DynaKube) (dynatrace.Option, error) {
	var proxyOption dynatrace.Option

	proxyURL, err := dk.Proxy(opts.ctx, apiReader)
	if err != nil {
		return proxyOption, err
	}

	proxyOption = dynatrace.WithProxy(proxyURL, dk.FF().GetNoProxy())

	return proxyOption, nil
}

func (opts *options) appendTrustedCerts(apiReader client.Reader, trustedCerts string, namespace string) error {
	if trustedCerts != "" {
		certs := &corev1.ConfigMap{}
		if err := apiReader.Get(opts.ctx, client.ObjectKey{Namespace: namespace, Name: trustedCerts}, certs); err != nil {
			return errors.WithMessage(err, "failed to get certificate configmap")
		}

		if certs.Data[dynakube.TrustedCAKey] == "" {
			return errors.New("failed to extract certificate configmap field: missing field certs")
		}

		opts.Opts = append(opts.Opts, dynatrace.WithCerts([]byte(certs.Data[dynakube.TrustedCAKey])))
	}

	return nil
}

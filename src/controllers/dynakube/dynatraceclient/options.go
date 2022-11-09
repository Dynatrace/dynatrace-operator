package dynatraceclient

import (
	"context"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
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

func newOptions(ctx context.Context) *options {
	return &options{
		Opts: []dtclient.Option{},
		ctx:  ctx,
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

func (opts *options) appendProxySettings(apiReader client.Reader, proxyEntry *dynatracev1beta1.DynaKubeProxy, namespace string) error {
	if proxyEntry == nil {
		return nil
	}

	proxyOption, err := opts.createProxyOption(apiReader, proxyEntry, namespace)
	if err != nil {
		return err
	}

	opts.Opts = append(opts.Opts, proxyOption)
	return nil
}

func (opts *options) createProxyOption(apiReader client.Reader, proxyEntry *dynatracev1beta1.DynaKubeProxy, namespace string) (dtclient.Option, error) {
	var proxyOption dtclient.Option

	if proxyEntry.ValueFrom != "" {
		proxyURL, err := opts.getProxyUrlFromSecret(apiReader, proxyEntry, namespace)
		if err != nil {
			return nil, err
		}

		proxyOption = dtclient.Proxy(proxyURL)
	} else if proxyEntry.Value != "" {
		proxyOption = dtclient.Proxy(proxyEntry.Value)
	}

	return proxyOption, nil
}

func (opts *options) getProxyUrlFromSecret(apiReader client.Reader, proxyEntry *dynatracev1beta1.DynaKubeProxy, namespace string) (string, error) {
	proxySecret := &corev1.Secret{}
	err := apiReader.Get(opts.ctx, client.ObjectKey{Name: proxyEntry.ValueFrom, Namespace: namespace}, proxySecret)

	if err != nil {
		return "", errors.WithMessage(err, "failed to get proxy secret")
	}

	proxyURL, err := kubeobjects.ExtractToken(proxySecret, dtclient.CustomProxySecretKey)
	if err != nil {
		return "", errors.WithMessage(err, "failed to extract proxy secret field")
	}

	return proxyURL, nil
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

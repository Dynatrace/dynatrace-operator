package version

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/url"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/dockerkeychain"
	"github.com/Dynatrace/dynatrace-operator/src/registry"
	containerv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ImageVersionFunc can fetch image information from img
type ImageVersionFunc func(
	ctx context.Context,
	apiReader client.Reader,
	registryClient registry.ImageGetter,
	dynakube *dynatracev1beta1.DynaKube,
	imageName string,
) (
	registry.ImageVersion,
	error,
)

var _ ImageVersionFunc = GetImageVersion

// GetImageVersion fetches image information for imageName
func GetImageVersion(
	ctx context.Context,
	apiReader client.Reader,
	registryClient registry.ImageGetter,
	dynakube *dynatracev1beta1.DynaKube,
	imageName string,
) (
	registry.ImageVersion,
	error,
) {
	keychain, err := dockerkeychain.NewDockerKeychain(ctx, apiReader, dynakube.PullSecretWithoutData())
	if err != nil {
		return registry.ImageVersion{}, errors.WithMessage(err, "failed to fetch pull secret")
	}

	transport, err := prepareTransport(ctx, apiReader, dynakube)
	if err != nil {
		return registry.ImageVersion{}, errors.WithMessage(err, "failed to prepare transport")
	}

	return registryClient.GetImageVersion(ctx, keychain, transport, imageName)
}

func PullImageInfo(
	ctx context.Context,
	apiReader client.Reader,
	registryClient registry.ImageGetter,
	dynakube *dynatracev1beta1.DynaKube,
	imageName string,
) (*containerv1.Image, error) {
	keychain, err := dockerkeychain.NewDockerKeychain(ctx, apiReader, dynakube.PullSecretWithoutData())
	if err != nil {
		return nil, errors.WithMessage(err, "failed to fetch pull secret")
	}

	transport, err := prepareTransport(ctx, apiReader, dynakube)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to prepare transport")
	}

	return registryClient.PullImageInfo(ctx, keychain, transport, imageName)
}

func prepareTransport(ctx context.Context, apiReader client.Reader, dynakube *dynatracev1beta1.DynaKube) (*http.Transport, error) {
	var err error
	var proxy string

	transport := http.DefaultTransport.(*http.Transport).Clone()

	if dynakube.HasProxy() {
		proxy, err = dynakube.Proxy(ctx, apiReader)
		if err != nil {
			return nil, err
		}
		proxyUrl, err := url.Parse(proxy)
		if err != nil {
			log.Info("invalid proxy spec", "proxy", proxy)
			return nil, errors.WithStack(err)
		}

		transport.Proxy = func(req *http.Request) (*url.URL, error) {
			return proxyUrl, nil
		}
	}

	if dynakube.Spec.TrustedCAs != "" {
		transport, err = AddCertificates(ctx, apiReader, transport, dynakube)
		if err != nil {
			return nil, errors.WithMessage(err, "failed adding trusted CAs to transport")
		}
	}
	return transport, nil
}

func AddCertificates(ctx context.Context, apiReader client.Reader, transport *http.Transport, dynakube *dynatracev1beta1.DynaKube) (*http.Transport, error) {
	trustedCAs, err := dynakube.TrustedCAs(ctx, apiReader)
	if err != nil {
		return transport, err
	}

	rootCAs := x509.NewCertPool()
	if ok := rootCAs.AppendCertsFromPEM(trustedCAs); !ok {
		log.Info("failed to append custom certs!")
	}
	if transport.TLSClientConfig == nil {
		transport.TLSClientConfig = &tls.Config{} // nolint:gosec
	}
	transport.TLSClientConfig.RootCAs = rootCAs

	return transport, nil
}

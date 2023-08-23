package version

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/dockerkeychain"
	"github.com/Dynatrace/dynatrace-operator/src/registry"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ImageVersionFunc can fetch image information from img
type ImageVersionFunc func(
	ctx context.Context,
	apiReader client.Reader,
	registryClient registry.ImageGetter,
	dynakube *dynatracev1beta1.DynaKube,
	imageName string,
	registryAuthPath string,
) (
	registry.ImageVersion,
	error,
)

var _ ImageVersionFunc = GetImageVersion

// GetImageVersion fetches image information for imageName
func GetImageVersion( //nolint:revive // argument-limit
	ctx context.Context,
	apiReader client.Reader,
	registryClient registry.ImageGetter,
	dynakube *dynatracev1beta1.DynaKube,
	imageName string,
	registryAuthPath string,
) (
	registry.ImageVersion,
	error,
) {
	var err error
	var proxy string

	transport := http.DefaultTransport.(*http.Transport).Clone()
	keychain := dockerkeychain.NewDockerKeychain()
	err = keychain.LoadDockerConfigFromSecret(ctx, apiReader, v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dynakube.PullSecret(),
			Namespace: dynakube.Namespace,
		},
	})
	if err != nil {
		log.Info("failed to fetch pull secret", "error", err)
	}

	if dynakube.HasProxy() {
		proxy, err = dynakube.Proxy(ctx, apiReader)
		if err != nil {
			return registry.ImageVersion{}, err
		}
		proxyUrl, err := url.Parse(proxy)
		if err != nil {
			log.Info("invalid proxy spec", "proxy", proxy)
			return registry.ImageVersion{}, err
		}

		transport.Proxy = func(req *http.Request) (*url.URL, error) {
			return proxyUrl, nil
		}
	}

	if dynakube.Spec.TrustedCAs != "" {
		transport, err = addCertificates(transport, dynakube, apiReader)
		if err != nil {
			return registry.ImageVersion{}, fmt.Errorf("addCertificates(): %w", err)
		}
	}

	return registryClient.GetImageVersion(ctx, keychain, transport, imageName)
}

func addCertificates(transport *http.Transport, dynakube *dynatracev1beta1.DynaKube, apiReader client.Reader) (*http.Transport, error) {
	trustedCAs, err := dynakube.TrustedCAs(context.TODO(), apiReader)
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

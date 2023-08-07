package version

import (
	"context"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"

	"github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/dockerconfig"
	"github.com/Dynatrace/dynatrace-operator/src/dockerkeychain"
	"github.com/Dynatrace/dynatrace-operator/src/registry"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/spf13/afero"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ImageVersionFunc can fetch image information from img
type ImageVersionFunc func(ctx context.Context, registryClient registry.ImageGetter, imageName string, dockerConfig *dockerconfig.DockerConfig) (registry.ImageVersion, error)

var _ ImageVersionFunc = GetImageVersion

// GetImageVersion fetches image information for imageName
func GetImageVersion(ctx context.Context, registryClient registry.ImageGetter, imageName string, dockerConfig *dockerconfig.DockerConfig) (registry.ImageVersion, error) {
	keychain := dockerkeychain.NewDockerKeychain(dockerConfig.RegistryAuthPath, afero.NewOsFs())
	transport := http.DefaultTransport.(*http.Transport).Clone()

	ref, err := name.ParseReference(imageName)
	if err != nil {
		return registry.ImageVersion{}, fmt.Errorf("parsing reference %q: %w", imageName, err)
	}

	if dockerConfig.Dynakube.HasProxy() {
		proxyUrl, err := url.Parse(dockerConfig.Dynakube.Spec.Proxy.Value)
		if err != nil {
			log.Info("invalid proxy spec", "proxy", dockerConfig.Dynakube.Spec.Proxy.Value)
			return registry.ImageVersion{}, err
		}

		transport.Proxy = func(req *http.Request) (*url.URL, error) {
			return proxyUrl, nil
		}
	}

	if dockerConfig.Dynakube.Spec.TrustedCAs != "" {
		transport, err = addCertificates(transport, dockerConfig.Dynakube, dockerConfig.ApiReader)
		if err != nil {
			return registry.ImageVersion{}, fmt.Errorf("addCertificates(): %w", err)
		}
	}

	return registryClient.GetImageVersion(ctx, ref, keychain, transport)
}

func addCertificates(transport *http.Transport, dynakube *dynakube.DynaKube, apiReader client.Reader) (*http.Transport, error) {
	trustedCAs, err := dynakube.TrustedCAs(context.TODO(), apiReader)
	if err != nil {
		return transport, err
	}

	rootCAs := x509.NewCertPool()
	if ok := rootCAs.AppendCertsFromPEM(trustedCAs); !ok {
		log.Info("failed to append custom certs!")
	}
	transport.TLSClientConfig.RootCAs = rootCAs

	return transport, nil
}

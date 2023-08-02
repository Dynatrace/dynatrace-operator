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
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/opencontainers/go-digest"
	"github.com/spf13/afero"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// VersionLabel is the name of the label used on ActiveGate-provided images.
	VersionLabel = "com.dynatrace.build-version"
)

type ImageVersion struct {
	Version string
	Digest  digest.Digest
}

// ImageVersionFunc can fetch image information from img
type ImageVersionFunc func(ctx context.Context, imageName string, dockerConfig *dockerconfig.DockerConfig) (ImageVersion, error)

var _ ImageVersionFunc = GetImageVersion

// GetImageVersion fetches image information for imageName
func GetImageVersion(ctx context.Context, imageName string, dockerConfig *dockerconfig.DockerConfig) (ImageVersion, error) {
	log.Info("GetImageVersion")

	ref, err := name.ParseReference(imageName)
	if err != nil {
		return ImageVersion{}, fmt.Errorf("parsing reference %q: %w", imageName, err)
	}

	log.Info("ref", "refName", ref.Name(), "refString", ref.String(), "refIdentifier", ref.Identifier(), "Context().RegistryStr()", ref.Context().RegistryStr(), "Context().Name()", ref.Context().Name(), "Context().Scheme()", ref.Context().Scheme())

	keychain := dockerkeychain.NewDockerKeychain(dockerConfig.RegistryAuthPath, afero.NewOsFs())

	transport := http.DefaultTransport.(*http.Transport).Clone()

	if dockerConfig.Dynakube.HasProxy() {
		proxyUrl, err := url.Parse(dockerConfig.Dynakube.Spec.Proxy.Value)
		if err != nil {
			log.Info("invalid proxy spec", "proxy", dockerConfig.Dynakube.Spec.Proxy.Value)
			return ImageVersion{}, err
		}

		transport.Proxy = func(req *http.Request) (*url.URL, error) {
			return proxyUrl, nil
		}
	}

	if dockerConfig.Dynakube.Spec.TrustedCAs != "" {
		transport, err = addCertificates(transport, dockerConfig.Dynakube, dockerConfig.ApiReader)
		if err != nil {
			return ImageVersion{}, fmt.Errorf("prepareCertificates(): %w", err)
		}
	}

	descriptor, err := remote.Get(ref, remote.WithContext(ctx), remote.WithAuthFromKeychain(keychain), remote.WithTransport(transport), remote.WithUserAgent("ao"))
	if err != nil {
		return ImageVersion{}, fmt.Errorf("getting reference %q: %w", imageName, err)
	}

	// TODO: does not work for indexes which contain schema v1 manifests
	img, err := descriptor.Image()
	if err != nil {
		return ImageVersion{}, fmt.Errorf("descriptor.Image(): %w", err)
	}
	dig, err := img.Digest()
	if err != nil {
		return ImageVersion{}, fmt.Errorf("img.Digest(): %w", err)
	}

	log.Info("go-containerregistry", "digest", dig)

	cf, err := img.ConfigFile()
	if err != nil {
		return ImageVersion{}, fmt.Errorf("img.ConfigFile: %w", err)
	}

	log.Info("go-containerregistry", "labels", cf.Config.Labels, "architecture", cf.Architecture, "author", cf.Author)

	return ImageVersion{
		Digest:  digest.Digest(dig.String()),
		Version: cf.Config.Labels[VersionLabel], // empty if unset
	}, nil
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

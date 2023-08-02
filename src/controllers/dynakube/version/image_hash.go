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

	proxyURL, err := prepareProxyURL(dockerConfig.Dynakube)
	if err != nil {
		return ImageVersion{}, fmt.Errorf("prepareProxyURL(): %w", err)
	}

	transport := prepareTransport(proxyURL)

	transport, err = prepareCertificates(transport, dockerConfig.Dynakube, dockerConfig.ApiReader)
	if err != nil {
		return ImageVersion{}, fmt.Errorf("prepareCertificates(): %w", err)
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

func prepareProxyURL(dynakube *dynakube.DynaKube) (*url.URL, error) {
	if !dynakube.HasProxy() {
		log.Info("proxy not defined")
		return nil, nil
	}

	proxyUrl, err := url.Parse(dynakube.Spec.Proxy.Value)
	if err != nil {
		log.Info("invalid proxy spec", "proxy", dynakube.Spec.Proxy.Value)
		return nil, err
	}

	log.Info("proxy spec", "proxy", dynakube.Spec.Proxy.Value, "proxyURL", proxyUrl.String(), "proxyURL.Host", proxyUrl.Host, "proxyURL.Port()", proxyUrl.Port())

	return proxyUrl, nil
}

func prepareTransport(proxyURL *url.URL) *http.Transport {
	transport := http.DefaultTransport.(*http.Transport).Clone()

	transport.Proxy = func(req *http.Request) (*url.URL, error) {
		proxyUrlName := ""
		if proxyURL != nil {
			proxyUrlName = proxyURL.String()
		}
		log.Info("via proxy", "proxyURL", proxyUrlName, "req.URL", req.URL.String(), "req.url.Scheme", req.URL.Scheme, "req.url.Host", req.URL.Host, "req.url.Port", req.URL.Port(), "req.User-Agent", req.Header.Get("User-Agent"))
		return proxyURL, nil
	}

	transport.OnProxyConnectResponse = func(ctx context.Context, proxyURL *url.URL, connectReq *http.Request, connectRes *http.Response) error {
		log.Info("OnProxyConnectResponse", "proxyURL", proxyURL, "connectReq.URL", connectReq.URL.String(), "connectReq.User-Agent", connectReq.Header.Get("User-Agent"), "connectRes", connectRes.Status, "connectRes.Request.URL", connectRes.Request.URL.String())
		return nil
	}

	return transport
}

func prepareCertificates(transport *http.Transport, dynakube *dynakube.DynaKube, apiReader client.Reader) (*http.Transport, error) {
	if dynakube.Spec.TrustedCAs != "" {
		trustedCAs, err := dynakube.TrustedCAs(context.TODO(), apiReader)
		if err != nil {
			return transport, err
		}

		rootCAs := x509.NewCertPool()
		if ok := rootCAs.AppendCertsFromPEM(trustedCAs); !ok {
			log.Info("failed to append custom certs!")
		}
		transport.TLSClientConfig.RootCAs = rootCAs
	}

	return transport, nil
}

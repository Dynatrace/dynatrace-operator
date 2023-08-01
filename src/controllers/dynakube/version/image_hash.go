package version

import (
	"context"
	"fmt"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dockerkeychain"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/spf13/afero"
	"net/http"
	"net/url"

	"github.com/Dynatrace/dynatrace-operator/src/dockerconfig"
	"github.com/containers/image/v5/image"
	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
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

// ImageVersionProxyFunc can fetch image trough proxy
type ImageVersionProxyFunc func(ctx context.Context, imageName string, dockerConfig *dockerconfig.DockerConfig, dynakube *dynatracev1beta1.DynaKube) (ImageVersion, error)

var _ ImageVersionFunc = GetImageVersion
var _ ImageVersionProxyFunc = GetImageVersionViaProxy

// GetImageVersion fetches image information for imageName
func GetImageVersion(ctx context.Context, imageName string, dockerConfig *dockerconfig.DockerConfig) (ImageVersion, error) {
	transportImageName := fmt.Sprintf("docker://%s", imageName)

	imageReference, err := alltransports.ParseImageName(transportImageName)
	if err != nil {
		return ImageVersion{}, errors.WithStack(err)
	}

	systemContext := dockerconfig.MakeSystemContext(imageReference.DockerReference(), dockerConfig)

	imageSource, err := imageReference.NewImageSource(context.TODO(), systemContext)
	if err != nil {
		return ImageVersion{}, errors.WithStack(err)
	}
	defer closeImageSource(imageSource)

	imageManifest, _, err := imageSource.GetManifest(context.TODO(), nil)
	if err != nil {
		return ImageVersion{}, errors.WithStack(err)
	}

	digest, err := manifest.Digest(imageManifest)
	if err != nil {
		return ImageVersion{}, errors.WithStack(err)
	}

	sourceImage, err := image.FromUnparsedImage(context.TODO(), systemContext, image.UnparsedInstance(imageSource, nil))
	if err != nil {
		return ImageVersion{}, errors.WithStack(err)
	}

	inspectedImage, err := sourceImage.Inspect(context.TODO())
	if err != nil {
		return ImageVersion{}, errors.WithStack(err)
	} else if inspectedImage == nil {
		return ImageVersion{}, errors.Errorf("could not inspect image: '%s'", transportImageName)
	}

	return ImageVersion{
		Digest:  digest,
		Version: inspectedImage.Labels[VersionLabel], // empty if unset
	}, nil
}

func GetImageVersionViaProxy(ctx context.Context, imageName string, dockerConfig *dockerconfig.DockerConfig, dynakube *dynatracev1beta1.DynaKube) (ImageVersion, error) {
	ref, err := name.ParseReference(imageName)
	if err != nil {
		return ImageVersion{}, fmt.Errorf("parsing reference %q: %w", imageName, err)
	}

	log.Info("ref", "refName", ref.Name(), "refString", ref.String(), "refIdentifier", ref.Identifier(), "Context().RegistryStr()", ref.Context().RegistryStr(), "Context().Name()", ref.Context().Name(), "Context().Scheme()", ref.Context().Scheme())

	// TODO: i'm not sure we use correct interface, we just need io.Reader interface instead of FS
	keychain := dockerkeychain.NewDockerKeychain(dockerConfig.RegistryAuthPath, afero.NewOsFs())

	var proxyUrl *url.URL
	if dynakube.HasProxy() {
		proxyUrl, err = url.Parse(dynakube.Spec.Proxy.Value)
		if err != nil {
			log.Info("invalid proxy spec", "proxy", dynakube.Spec.Proxy.Value)
			return ImageVersion{}, err
		}
		log.Info("proxy spec", "proxy", dynakube.Spec.Proxy.Value, "proxyURL", proxyUrl.String(), "proxyURL.Host", proxyUrl.Host, "proxyURL.Port()", proxyUrl.Port())
	} else {
		log.Info("proxy not defined")
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = func(req *http.Request) (*url.URL, error) {
		proxyUrlName := ""
		if proxyUrl != nil {
			proxyUrlName = proxyUrl.String()
		}
		log.Info("via proxy", "proxyURL", proxyUrlName, "req.URL", req.URL.String(), "req.url.Scheme", req.URL.Scheme, "req.url.Host", req.URL.Host, "req.url.Port", req.URL.Port(), "req.User-Agent", req.Header.Get("User-Agent"))
		return proxyUrl, nil
	}
	transport.OnProxyConnectResponse = func(ctx context.Context, proxyURL *url.URL, connectReq *http.Request, connectRes *http.Response) error {
		log.Info("OnProxyConnectResponse", "proxyURL", proxyURL, "connectReq.URL", connectReq.URL.String(), "connectReq.User-Agent", connectReq.Header.Get("User-Agent"), "connectRes", connectRes.Status, "connectRes.Request.URL", connectRes.Request.URL.String())
		return nil
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

func closeImageSource(source types.ImageSource) {
	if source != nil {
		// Swallow error
		_ = source.Close()
	}
}

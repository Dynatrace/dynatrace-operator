package registry

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"

	"github.com/Dynatrace/dynatrace-operator/src/dockerkeychain"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	containerv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ImageGetter interface {
	GetImageVersion(ctx context.Context, imageName string) (ImageVersion, error)
	PullImageInfo(ctx context.Context, imageName string) (*containerv1.Image, error)
}

type ImageVersion struct {
	Version string
	Digest  digest.Digest
}

type Client struct {
	keychain  authn.Keychain
	transport *http.Transport
}

const (
	// VersionLabel is the name of the label used on ActiveGate-provided images.
	VersionLabel    = "com.dynatrace.build-version"
	DigestDelimiter = "@"
)

func NewClient(ctx context.Context, apiReader client.Reader, keyChainSecret *corev1.Secret, proxy string, trustedCAs []byte) (*Client, error) {
	var keychain authn.Keychain
	var err error
	if keyChainSecret != nil {
		keychain, err = dockerkeychain.NewDockerKeychain(ctx, apiReader, *keyChainSecret)
		if err != nil {
			return nil, errors.WithMessage(err, "failed to fetch pull secret")
		}
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport, err = PrepareTransport(transport, proxy, trustedCAs)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to prepare transport")
	}

	return &Client{keychain: keychain, transport: transport}, nil
}

func (c *Client) GetImageVersion(ctx context.Context, imageName string) (ImageVersion, error) {
	ref, err := name.ParseReference(imageName)
	if err != nil {
		return ImageVersion{}, fmt.Errorf("parsing reference %q: %w", imageName, err)
	}

	options := []remote.Option{
		remote.WithContext(ctx),
		remote.WithTransport(c.transport),
	}
	if c.keychain != nil {
		options = append(options, remote.WithAuthFromKeychain(c.keychain))
	}

	descriptor, err := remote.Get(ref, options...)
	if err != nil {
		return ImageVersion{}, fmt.Errorf("getting reference %q: %w", ref, err)
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
	cf, err := img.ConfigFile()
	if err != nil {
		return ImageVersion{}, fmt.Errorf("img.ConfigFile: %w", err)
	}

	return ImageVersion{
		Digest:  digest.Digest(dig.String()),
		Version: cf.Config.Labels[VersionLabel], // empty if unset
	}, nil
}

func (c *Client) PullImageInfo(ctx context.Context, imageName string) (*containerv1.Image, error) {
	ref, err := name.ParseReference(imageName)
	if err != nil {
		return nil, errors.WithMessagef(err, "parsing reference %q:", imageName)
	}

	image, err := remote.Image(ref, remote.WithContext(ctx), remote.WithAuthFromKeychain(c.keychain), remote.WithTransport(c.transport))
	if err != nil {
		return nil, errors.WithMessagef(err, "getting image %q", imageName)
	}

	return &image, nil
}

<<<<<<< HEAD
func BuildImageIDWithTagAndDigest(taggedRef name.Tag, digest digest.Digest) string {
	return fmt.Sprintf("%s%s%s", taggedRef.String(), DigestDelimiter, digest.String())
}

func PrepareTransport(ctx context.Context, apiReader client.Reader, transport *http.Transport, dynakube *dynatracev1beta1.DynaKube) (*http.Transport, error) {
	var err error
	var proxy string

	if dynakube.HasProxy() {
		proxy, err = dynakube.Proxy(ctx, apiReader)
		if err != nil {
			return nil, err
		}
		proxyUrl, err := url.Parse(proxy)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		transport.Proxy = func(req *http.Request) (*url.URL, error) {
			return proxyUrl, nil
		}
||||||| parent of 9ecb6aba (fixup! tests)
func PrepareTransport(ctx context.Context, apiReader client.Reader, transport *http.Transport, dynakube *dynatracev1beta1.DynaKube) (*http.Transport, error) {
	var err error
	var proxy string

	if dynakube.HasProxy() {
		proxy, err = dynakube.Proxy(ctx, apiReader)
		if err != nil {
			return nil, err
		}
		proxyUrl, err := url.Parse(proxy)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		transport.Proxy = func(req *http.Request) (*url.URL, error) {
			return proxyUrl, nil
		}
=======
func addProxy(transport *http.Transport, proxy string) (*http.Transport, error) {
	proxyUrl, err := url.Parse(proxy)
	if err != nil {
		return nil, errors.WithStack(err)
>>>>>>> 9ecb6aba (fixup! tests)
	}

	transport.Proxy = func(req *http.Request) (*url.URL, error) {
		return proxyUrl, nil
	}
	return transport, nil
}

func addCertificates(transport *http.Transport, trustedCAs []byte) (*http.Transport, error) {
	rootCAs := x509.NewCertPool()
	if ok := rootCAs.AppendCertsFromPEM(trustedCAs); !ok {
		return nil, errors.New("failed to append custom certs")
	}
	if transport.TLSClientConfig == nil {
		transport.TLSClientConfig = &tls.Config{} // nolint:gosec
	}
	transport.TLSClientConfig.RootCAs = rootCAs

	return transport, nil
}

// PrepareTransport creates default http transport and add proxy or trustedCAs if any
func PrepareTransport(transport *http.Transport, proxy string, trustedCAs []byte) (*http.Transport, error) {
	var err error

	if proxy != "" {
		transport, err = addProxy(transport, proxy)
		if err != nil {
			return nil, errors.WithMessage(err, "failed to add proxy to default transport")
		}
	}

	if len(trustedCAs) > 0 {
		transport, err = addCertificates(transport, trustedCAs)
		if err != nil {
			return nil, err
		}
	}

	return transport, nil
}

package registry

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/arch"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/oci/dockerkeychain"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	containerv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
	"golang.org/x/net/http/httpproxy"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ClientBuilder func(options ...func(*Client)) (ImageGetter, error)

type ImageGetter interface {
	GetImageVersion(ctx context.Context, imageName string) (ImageVersion, error)
	PullImageInfo(ctx context.Context, imageName string) (*containerv1.Image, error)
}

type ImageVersion struct {
	Version string
	Digest  digest.Digest
	Type    string
}

type Client struct {
	ctx            context.Context
	apiReader      client.Reader
	keychain       authn.Keychain
	keyChainSecret *corev1.Secret
	transport      *http.Transport
}

const (
	// TypeLabel is the name of the label that indicates if image is immutable or mutable.
	TypeLabel = "com.dynatrace.type"
	// VersionLabel is the name of the label used on ActiveGate-provided images.
	VersionLabel    = "com.dynatrace.build-version"
	DigestDelimiter = "@"
)

func WithContext(ctx context.Context) func(*Client) {
	return func(c *Client) {
		c.ctx = ctx
	}
}

func WithAPIReader(apiReader client.Reader) func(*Client) {
	return func(c *Client) {
		c.apiReader = apiReader
	}
}

func WithKeyChainSecret(keyChainSecret *corev1.Secret) func(*Client) {
	return func(c *Client) {
		c.keyChainSecret = keyChainSecret
	}
}

func WithTransport(transport *http.Transport) func(*Client) {
	return func(c *Client) {
		c.transport = transport
	}
}

func NewClient(options ...func(*Client)) (ImageGetter, error) {
	var err error

	c := &Client{}
	for _, opt := range options {
		opt(c)
	}

	if c.keyChainSecret != nil {
		keychain, err := dockerkeychain.NewDockerKeychain(c.ctx, c.apiReader, *c.keyChainSecret)
		if err != nil {
			return nil, errors.WithMessage(err, "failed to fetch pull secret")
		}

		c.keychain = keychain
	}

	if err != nil {
		return nil, errors.WithMessage(err, "failed to prepare transport")
	}

	return c, nil
}

var _ ClientBuilder = NewClient

func (c *Client) GetImageVersion(ctx context.Context, imageName string) (ImageVersion, error) {
	ref, err := name.ParseReference(imageName)
	if err != nil {
		return ImageVersion{}, errors.WithMessagef(err, "parsing reference %q", imageName)
	}

	options := []remote.Option{
		remote.WithContext(ctx),
		remote.WithTransport(c.transport),
		remote.WithPlatform(arch.ImagePlatform),
	}
	if c.keychain != nil {
		options = append(options, remote.WithAuthFromKeychain(c.keychain))
	}

	descriptor, err := remote.Get(ref, options...)
	if err != nil {
		return ImageVersion{}, errors.WithMessagef(err, "getting reference %q", ref)
	}

	// TODO: does not work for indexes which contain schema v1 manifests
	img, err := descriptor.Image()
	if err != nil {
		return ImageVersion{}, errors.WithMessagef(err, "descriptor.Image()")
	}

	// use image digest as a fallback
	digestFn := img.Digest

	// try to get image manifest to cover multi arch images
	imageIndex, err := descriptor.ImageIndex()
	if err == nil {
		digestFn = imageIndex.Digest
	}

	dig, err := digestFn()
	if err != nil {
		return ImageVersion{}, errors.WithMessagef(err, "could not get image digest")
	}

	cf, err := img.ConfigFile()
	if err != nil {
		return ImageVersion{}, errors.WithMessagef(err, "img.ConfigFile")
	}

	return ImageVersion{
		Digest:  digest.Digest(dig.String()),
		Version: cf.Config.Labels[VersionLabel], // empty if unset
		Type:    cf.Config.Labels[TypeLabel],    // empty if unset
	}, nil
}

func (c *Client) PullImageInfo(ctx context.Context, imageName string) (*containerv1.Image, error) {
	ref, err := name.ParseReference(imageName)
	if err != nil {
		return nil, errors.WithMessagef(err, "parsing reference %q:", imageName)
	}

	image, err := remote.Image(ref,
		remote.WithContext(ctx),
		remote.WithAuthFromKeychain(c.keychain),
		remote.WithTransport(c.transport),
		remote.WithPlatform(arch.ImagePlatform),
	)
	if err != nil {
		return nil, errors.WithMessagef(err, "getting image %q", imageName)
	}

	return &image, nil
}

func BuildImageIDWithTagAndDigest(taggedRef name.Tag, digest digest.Digest) string {
	return fmt.Sprintf("%s%s%s", taggedRef.String(), DigestDelimiter, digest.String())
}

func addProxy(transport *http.Transport, proxy string, noProxy string) (*http.Transport, error) {
	proxyURL, err := url.Parse(proxy)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	proxyConfig := httpproxy.Config{
		HTTPProxy:  proxyURL.String(),
		HTTPSProxy: proxyURL.String(),
		NoProxy:    noProxy,
	}
	transport.Proxy = proxyWrapper(proxyConfig)

	return transport, nil
}

func proxyWrapper(proxyConfig httpproxy.Config) func(req *http.Request) (*url.URL, error) {
	return func(req *http.Request) (*url.URL, error) {
		return proxyConfig.ProxyFunc()(req.URL)
	}
}

func addCertificates(transport *http.Transport, trustedCAs []byte) (*http.Transport, error) {
	rootCAs, err := x509.SystemCertPool()
	if err != nil {
		return nil, errors.Wrap(err, "couldn't read system certificates")
	}

	if ok := rootCAs.AppendCertsFromPEM(trustedCAs); !ok {
		return nil, errors.New("failed to append custom certs")
	}

	if transport.TLSClientConfig == nil {
		transport.TLSClientConfig = &tls.Config{} //nolint:gosec
	}

	transport.TLSClientConfig.RootCAs = rootCAs

	return transport, nil
}

func addSkipCertCheck(transport *http.Transport, skipCertCheck bool) *http.Transport {
	if transport.TLSClientConfig == nil {
		transport.TLSClientConfig = &tls.Config{} //nolint:gosec
	}

	transport.TLSClientConfig.InsecureSkipVerify = skipCertCheck

	return transport
}

// PrepareTransportForDynaKube creates default http transport and add proxy or trustedCAs if any
func PrepareTransportForDynaKube(ctx context.Context, apiReader client.Reader, transport *http.Transport, dk *dynakube.DynaKube) (*http.Transport, error) {
	var (
		proxy      string
		trustedCAs []byte
		err        error
	)

	if dk.HasProxy() {
		proxy, err = dk.Proxy(ctx, apiReader)
		if err != nil {
			return nil, err
		}
	}

	if dk.Spec.TrustedCAs != "" {
		trustedCAs, err = dk.TrustedCAs(ctx, apiReader)
		if err != nil {
			return nil, err
		}
	}

	if proxy != "" {
		transport, err = addProxy(transport, proxy, dk.FF().GetNoProxy())
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

	transport = addSkipCertCheck(transport, dk.Spec.SkipCertCheck)

	return transport, nil
}

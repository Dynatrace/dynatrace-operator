package registry

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
)

type ImageGetter interface {
	GetImageVersion(ctx context.Context, keychain authn.Keychain, transport *http.Transport, imageName string) (ImageVersion, error)
	PullImageInfo(ctx context.Context, keychain authn.Keychain, transport *http.Transport, imageName string) (*v1.Image, error)
}

type ImageVersion struct {
	Version string
	Digest  digest.Digest
}

type Client struct{}

const (
	// VersionLabel is the name of the label used on ActiveGate-provided images.
	VersionLabel = "com.dynatrace.build-version"
)

func NewClient() *Client {
	return &Client{}
}

func (c *Client) GetImageVersion(ctx context.Context, keychain authn.Keychain, transport *http.Transport, imageName string) (ImageVersion, error) {
	ref, err := name.ParseReference(imageName)
	if err != nil {
		return ImageVersion{}, fmt.Errorf("parsing reference %q: %w", imageName, err)
	}

	options := []remote.Option{
		remote.WithContext(ctx),
		remote.WithTransport(transport),
	}
	if keychain != nil {
		options = append(options, remote.WithAuthFromKeychain(keychain))
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

func (c *Client) PullImageInfo(ctx context.Context, keychain authn.Keychain, transport *http.Transport, imageName string) (*v1.Image, error) {
	ref, err := name.ParseReference(imageName)
	if err != nil {
		return nil, errors.WithMessagef(err, "parsing reference %q:", imageName)
	}

	image, err := remote.Image(ref, remote.WithContext(ctx), remote.WithAuthFromKeychain(keychain), remote.WithTransport(transport))
	if err != nil {
		return nil, errors.WithMessagef(err, "getting image %q", imageName)
	}

	return &image, nil
}

package registry

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/opencontainers/go-digest"
)

type Client interface {
	GetImageVersion(ctx context.Context, ref name.Reference, keychain authn.Keychain, transport *http.Transport) (ImageVersion, error)
}

type ImageVersion struct {
	Version string
	Digest  digest.Digest
}

type GoContainerRegistryClient struct{}

const (
	// VersionLabel is the name of the label used on ActiveGate-provided images.
	VersionLabel = "com.dynatrace.build-version"
)

func (r *GoContainerRegistryClient) GetImageVersion(ctx context.Context, ref name.Reference, keychain authn.Keychain, transport *http.Transport) (ImageVersion, error) {
	descriptor, err := remote.Get(ref, remote.WithContext(ctx), remote.WithAuthFromKeychain(keychain), remote.WithTransport(transport))
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

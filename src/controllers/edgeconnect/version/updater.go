package version

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/src/api/status"
	edgeconnectv1alpha1 "github.com/Dynatrace/dynatrace-operator/src/api/v1alpha1/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/src/dockerkeychain"
	"github.com/Dynatrace/dynatrace-operator/src/registry"
	"github.com/Dynatrace/dynatrace-operator/src/timeprovider"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type updater struct {
	edgeConnect    *edgeconnectv1alpha1.EdgeConnect
	apiReader      client.Reader
	timeProvider   *timeprovider.Provider
	registryClient registry.ImageGetter
}

var _ versionStatusUpdater = updater{}

func newUpdater(
	apiReader client.Reader,
	timeprovider *timeprovider.Provider,
	registryClient registry.ImageGetter,
	edgeConnect *edgeconnectv1alpha1.EdgeConnect,
) *updater {
	return &updater{
		edgeConnect:    edgeConnect,
		apiReader:      apiReader,
		timeProvider:   timeprovider,
		registryClient: registryClient,
	}
}

func (u updater) RequiresReconcile() bool {
	version := u.edgeConnect.Status.Version

	isRequestOutdated := u.timeProvider.IsOutdated(version.LastProbeTimestamp, edgeconnectv1alpha1.DefaultMinRequestThreshold)
	didCustomImageChange := !strings.HasPrefix(version.ImageID, u.edgeConnect.Image())

	if didCustomImageChange || version.ImageID == "" {
		return true
	}
	return isRequestOutdated && u.IsAutoUpdateEnabled()
}

func (u updater) Update(ctx context.Context) error {
	var err error
	defer func() {
		if err == nil {
			u.Target().LastProbeTimestamp = u.timeProvider.Now()
		}
	}()

	image := u.edgeConnect.Image()

	transport := http.DefaultTransport.(*http.Transport).Clone()

	keychain, err := dockerkeychain.NewDockerKeychain(ctx, u.apiReader, u.edgeConnect.PullSecretWithoutData())
	if err != nil {
		return err
	}

	imageVersion, err := u.registryClient.GetImageVersion(ctx, keychain, transport, image)
	if err != nil {
		return err
	}
	imageID, err := u.combineImageWithDigest(imageVersion.Digest)
	if err != nil {
		return err
	}

	target := u.Target()
	target.ImageID = imageID

	if u.edgeConnect.IsCustomImage() {
		target.Source = status.CustomImageVersionSource
	} else {
		target.Source = status.PublicRegistryVersionSource
	}

	return nil
}

func (u updater) combineImageWithDigest(digest digest.Digest) (string, error) {
	imageRef, err := name.ParseReference(u.edgeConnect.Image())
	if err != nil {
		return "", errors.WithStack(err)
	}
	if taggedRef, ok := imageRef.(name.Tag); ok {
		canonRef := registry.BuildImageIDWithTagAndDigest(taggedRef, digest)
		if err != nil {
			return "", errors.WithStack(err)
		}
		return canonRef, nil
	}
	return "", fmt.Errorf("wrong image reference format")
}

func (u updater) Name() string {
	return "edgeconnect"
}

func (u updater) Target() *status.VersionStatus {
	return &u.edgeConnect.Status.Version
}

func (u updater) IsAutoUpdateEnabled() bool {
	return u.edgeConnect.Spec.AutoUpdate
}

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
	"github.com/containers/image/v5/docker/reference"
	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type edgeConnectUpdater struct {
	edgeConnect    *edgeconnectv1alpha1.EdgeConnect
	apiReader      client.Reader
	timeProvider   *timeprovider.Provider
	registryClient registry.ImageGetter
}

var _ versionStatusUpdater = edgeConnectUpdater{}

func newEdgeConnectUpdater(
	edgeConnect *edgeconnectv1alpha1.EdgeConnect,
	apiReader client.Reader,
	timeprovider *timeprovider.Provider,
) *edgeConnectUpdater {
	return &edgeConnectUpdater{
		edgeConnect:    edgeConnect,
		apiReader:      apiReader,
		timeProvider:   timeprovider,
		registryClient: registry.NewClient(),
	}
}

func (updater edgeConnectUpdater) RequiresReconcile() bool {
	version := updater.edgeConnect.Status.Version

	isRequestOutdated := updater.timeProvider.IsOutdated(version.LastProbeTimestamp, edgeconnectv1alpha1.DefaultMinRequestThreshold)
	didCustomImageChange := !strings.HasPrefix(version.ImageID, updater.edgeConnect.Image())

	if didCustomImageChange || version.ImageID == "" {
		return true
	}
	return isRequestOutdated && updater.IsAutoUpdateEnabled()
}

func (updater edgeConnectUpdater) Update(ctx context.Context) error {
	var err error
	defer func() {
		if err == nil {
			updater.Target().LastProbeTimestamp = updater.timeProvider.Now()
		}
	}()

	image := updater.edgeConnect.Image()

	transport := http.DefaultTransport.(*http.Transport).Clone()

	keychain, err := dockerkeychain.NewDockerKeychain(ctx, updater.apiReader, updater.edgeConnect.PullSecretWithoutData())
	if err != nil {
		return err
	}

	imageVersion, err := updater.registryClient.GetImageVersion(ctx, keychain, transport, image)
	if err != nil {
		return err
	}
	imageID, err := updater.combineImageWithDigest(imageVersion.Digest)
	if err != nil {
		return err
	}

	target := updater.Target()
	target.ImageID = imageID

	if updater.edgeConnect.IsCustomImage() {
		target.Source = status.CustomImageVersionSource
	} else {
		target.Source = status.PublicRegistryVersionSource
	}

	return nil
}

func (updater edgeConnectUpdater) combineImageWithDigest(digest digest.Digest) (string, error) {
	imageRef, err := reference.Parse(updater.edgeConnect.Image())
	if err != nil {
		return "", errors.WithStack(err)
	}
	if taggedRef, ok := imageRef.(reference.NamedTagged); ok {
		canonRef, err := reference.WithDigest(taggedRef, digest)
		if err != nil {
			return "", errors.WithStack(err)
		}
		return canonRef.String(), nil
	}
	return "", fmt.Errorf("image reference wrongly formatted")
}

func (updater edgeConnectUpdater) Name() string {
	return "edgeconnect"
}

func (updater edgeConnectUpdater) Target() *status.VersionStatus {
	return &updater.edgeConnect.Status.Version
}

func (updater edgeConnectUpdater) IsAutoUpdateEnabled() bool {
	return updater.edgeConnect.Spec.AutoUpdate
}

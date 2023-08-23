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
<<<<<<< HEAD
	edgeConnect    *edgeconnectv1alpha1.EdgeConnect
	apiReader      client.Reader
	timeProvider   *timeprovider.Provider
	registryClient registry.ImageGetter
||||||| parent of 4c7e4959 (Update unit tests)
	edgeConnect  *edgeconnectv1alpha1.EdgeConnect
	apiReader    client.Reader
	timeProvider *timeprovider.Provider
=======
	edgeConnect    *edgeconnectv1alpha1.EdgeConnect
	apiReader      client.Reader
	timeProvider   *timeprovider.Provider
	dockerKeyChain *dockerkeychain.DockerKeychain
	registryClient registry.ImageGetter
>>>>>>> 4c7e4959 (Update unit tests)
}

var _ versionStatusUpdater = edgeConnectUpdater{}

func newEdgeConnectUpdater(
	edgeConnect *edgeconnectv1alpha1.EdgeConnect,
	apiReader client.Reader,
	timeprovider *timeprovider.Provider,
) *edgeConnectUpdater {
	return &edgeConnectUpdater{
<<<<<<< HEAD
		edgeConnect:    edgeConnect,
		apiReader:      apiReader,
		timeProvider:   timeprovider,
		registryClient: registry.NewClient(),
||||||| parent of 4c7e4959 (Update unit tests)
		edgeConnect:  edgeConnect,
		apiReader:    apiReader,
		timeProvider: timeprovider,
=======
		edgeConnect:    edgeConnect,
		apiReader:      apiReader,
		timeProvider:   timeprovider,
		dockerKeyChain: dockerkeychain.NewDockerKeychain(),
		registryClient: registry.NewClient(),
>>>>>>> 4c7e4959 (Update unit tests)
	}
}

func (updater edgeConnectUpdater) RequiresReconcile() bool {
	version := updater.edgeConnect.Status.Version

<<<<<<< HEAD
	isRequestOutdated := updater.timeProvider.IsOutdated(version.LastProbeTimestamp, edgeconnectv1alpha1.DefaultMinRequestThreshold)
	didCustomImageChange := !strings.HasPrefix(version.ImageID, updater.edgeConnect.Image())

	if didCustomImageChange || version.ImageID == "" {
||||||| parent of 4c7e4959 (Update unit tests)
	if didCustomImageChange || updater.edgeConnect.Status.Version.ImageID == "" {
=======
	isRequestOutdated := updater.timeProvider.IsOutdated(version.LastProbeTimestamp, DefaultMinRequestThreshold)
	didCustomImageChange := !strings.HasPrefix(version.ImageID, updater.edgeConnect.Image())

	if didCustomImageChange || version.ImageID == "" {
>>>>>>> 4c7e4959 (Update unit tests)
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

<<<<<<< HEAD
	keychain, err := dockerkeychain.NewDockerKeychain(ctx, updater.apiReader, updater.edgeConnect.PullSecretWithoutData())
||||||| parent of 4c7e4959 (Update unit tests)
	dockerKeyChain, err := dockerkeychain.NewDockerKeychain(ctx, updater.apiReader, corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      updater.edgeConnect.Spec.CustomPullSecret,
			Namespace: updater.edgeConnect.Namespace,
		},
	})
=======
	err = updater.dockerKeyChain.LoadDockerConfigFromSecret(ctx, updater.apiReader, corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      updater.edgeConnect.Spec.CustomPullSecret,
			Namespace: updater.edgeConnect.Namespace,
		},
	})
>>>>>>> 4c7e4959 (Update unit tests)
	if err != nil {
		return err
	}

<<<<<<< HEAD
	imageVersion, err := updater.registryClient.GetImageVersion(ctx, keychain, transport, image)
||||||| parent of 4c7e4959 (Update unit tests)
	imageVersion, err := registry.NewClient().GetImageVersion(ctx, dockerKeyChain, transport, image)
=======
	imageVersion, err := updater.registryClient.GetImageVersion(ctx, updater.dockerKeyChain, transport, image)
>>>>>>> 4c7e4959 (Update unit tests)
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

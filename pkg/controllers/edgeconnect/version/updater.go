package version

import (
	"context"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	edgeconnectv1alpha1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/pkg/oci/registry"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
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
	target := u.Target()

	if !u.edgeConnect.IsCustomImage() {
		log.Debug("EdgeConnect public registry image used")
		imageVersion, err := u.registryClient.GetImageVersion(ctx, image)
		if err != nil {
			return err
		}

		image, err = u.combineImageWithDigest(imageVersion.Digest)

		if err != nil {
			return err
		}
		target.Source = status.PublicRegistryVersionSource
	} else {
		log.Debug("EdgeConnect custom image used")
		target.Source = status.CustomImageVersionSource
	}

	target.ImageID = image

	return nil
}

func (u updater) combineImageWithDigest(digest digest.Digest) (string, error) {
	imageRef, err := name.ParseReference(u.edgeConnect.Image())
	if err != nil {
		log.Debug("unable to parse EdgeConnect image reference", "error", err.Error())
		return "", errors.WithStack(err)
	}

	if taggedRef, ok := imageRef.(name.Tag); ok {
		canonRef := registry.BuildImageIDWithTagAndDigest(taggedRef, digest)
		log.Debug("canonical image reference", "reference", canonRef)

		return canonRef, nil
	}

	log.Debug("wrong image reference format", "reference", imageRef.String())

	return "", errors.New("wrong image reference format")
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

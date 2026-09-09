// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package version

import (
	"context"
	"strings"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	dtimage "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/oci/registry"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const minRequestThreshold = 15 * time.Minute

type updater struct {
	edgeConnect            *edgeconnect.EdgeConnect
	apiReader              client.Reader
	timeProvider           *timeprovider.Provider
	imageClientProvider    ImageClientProvider
	registryClientProvider RegistryClientProvider
}

var _ versionStatusUpdater = updater{}

func newUpdater(
	apiReader client.Reader,
	timeprovider *timeprovider.Provider,
	imageClientProvider ImageClientProvider,
	registryClientProvider RegistryClientProvider,
	ec *edgeconnect.EdgeConnect,
) *updater {
	return &updater{
		edgeConnect:            ec,
		apiReader:              apiReader,
		timeProvider:           timeprovider,
		imageClientProvider:    imageClientProvider,
		registryClientProvider: registryClientProvider,
	}
}

func (u updater) determineSource() status.VersionSource {
	if u.edgeConnect.IsCustomImage() {
		return status.CustomImageVersionSource
	}

	return status.PublicRegistryVersionSource
}

func (u updater) RequiresReconcile() bool {
	version := u.edgeConnect.Status.Version

	if version.ImageID == "" {
		return true
	}

	// switching between a custom image and the public registry has to be applied right away
	if version.Source != u.determineSource() {
		return true
	}

	if u.edgeConnect.IsCustomImage() {
		// a custom image is taken over as-is, so any change of the image field has to be applied right away
		return !strings.HasPrefix(version.ImageID, u.edgeConnect.Image())
	}

	// a different public registry has to be applied right away, otherwise the image is only
	// refreshed if auto update is enabled
	if u.hasPublicRegistryChanged(version.ImageID) {
		return true
	}

	return u.timeProvider.IsOutdated(version.LastProbeTimestamp, minRequestThreshold) && u.IsAutoUpdateEnabled()
}

// hasPublicRegistryChanged reports whether the image in the status was pulled from a registry other
// than the one currently requested. Without an override the registry is chosen by fleet management,
// so there is nothing to compare the status against.
func (u updater) hasPublicRegistryChanged(imageID string) bool {
	registryOverride := u.edgeConnect.Spec.PublicRegistryOverride
	if registryOverride == "" {
		return false
	}

	return !strings.HasPrefix(imageID, registryOverride+"/")
}

func (u updater) Update(ctx context.Context) error {
	log := logd.FromContext(ctx)
	currentSource := u.determineSource()

	var err error

	defer func() {
		if err == nil {
			u.Target().Source = currentSource
			u.Target().LastProbeTimestamp = u.timeProvider.Now()
		}
	}()

	if currentSource == status.CustomImageVersionSource {
		log.Debug("updating version status according to custom image")
		setImageIDToCustomImage(ctx, u.Target(), u.edgeConnect.Image())

		return nil
	}

	log.Debug("updating version status according to the public registry")

	err = u.usePublicRegistry(ctx)

	return err
}

// usePublicRegistry resolves the image through fleet management and falls back to querying the
// public OCI registry directly, because fleet management is not generally available yet.
func (u updater) usePublicRegistry(ctx context.Context) error {
	log := logd.FromContext(ctx)

	imageInfo, fleetErr := u.latestImageInfo(ctx)

	switch {
	case fleetErr != nil:
		log.Info("fleet management image resolution failed", "error", fleetErr)
	case imageInfo == nil:
		fleetErr = errors.New("fleet management returned no image")

		log.Info("fleet management returned no image")
	default:
		setImageFromImageInfo(ctx, u.Target(), imageInfo)

		return nil
	}

	// the fallback can only resolve the default EdgeConnect image, so taking it would silently
	// ignore the registry the user explicitly asked for
	if registryOverride := u.edgeConnect.Spec.PublicRegistryOverride; registryOverride != "" {
		log.Info("no OCI registry fallback because a public registry override is set", "registry", registryOverride)

		return errors.WithMessagef(fleetErr, "cannot resolve the EdgeConnect image from the overridden registry %q", registryOverride)
	}

	log.Info("falling back to the OCI registry")

	if err := u.useOCIRegistry(ctx); err != nil {
		return errors.WithMessagef(err, "OCI registry fallback failed after fleet management error (%v)", fleetErr)
	}

	return nil
}

func (u updater) latestImageInfo(ctx context.Context) (*dtimage.Info, error) {
	imagesClient, err := u.imageClientProvider(ctx)
	if err != nil {
		return nil, err
	}

	return imagesClient.GetComponentLatestInfo(ctx, dtimage.EdgeConnect, u.edgeConnect.Spec.PublicRegistryOverride)
}

func (u updater) useOCIRegistry(ctx context.Context) error {
	registryClient, err := u.registryClientProvider(ctx)
	if err != nil {
		return err
	}

	imageVersion, err := registryClient.GetImageVersion(ctx, u.edgeConnect.Image())
	if err != nil {
		return err
	}

	imageID, err := u.combineImageWithDigest(ctx, imageVersion.Digest)
	if err != nil {
		return err
	}

	setImageFromOCIRegistry(ctx, u.Target(), imageID, imageVersion.Version)

	return nil
}

func (u updater) combineImageWithDigest(ctx context.Context, dig digest.Digest) (string, error) {
	log := logd.FromContext(ctx)

	imageRef, err := name.ParseReference(u.edgeConnect.Image())
	if err != nil {
		log.Debug("unable to parse EdgeConnect image reference")

		return "", errors.WithStack(err)
	}

	if taggedRef, ok := imageRef.(name.Tag); ok {
		canonRef := registry.BuildImageIDWithTagAndDigest(taggedRef, dig)
		log.Debug("canonical image reference", "reference", canonRef)

		return canonRef, nil
	}

	log.Debug("wrong image reference format", "reference", imageRef.String())

	return "", errors.New("wrong image reference format")
}

func setImageFromImageInfo(ctx context.Context, target *status.VersionStatus, imageInfo *dtimage.Info) {
	log := logd.FromContext(ctx)
	oldImageID := target.ImageID

	target.ImageID = imageInfo.URI
	target.Version = imageInfo.Tag

	log.Info("updated image version info",
		"oldImageID", oldImageID,
		"newImageID", target.ImageID,
		"version", target.Version)
}

func setImageFromOCIRegistry(ctx context.Context, target *status.VersionStatus, imageID string, version string) {
	log := logd.FromContext(ctx)
	oldImageID := target.ImageID

	target.ImageID = imageID
	target.Version = version

	log.Info("updated image version info from the OCI registry",
		"oldImageID", oldImageID,
		"newImageID", target.ImageID,
		"version", target.Version)
}

func setImageIDToCustomImage(ctx context.Context, target *status.VersionStatus, imageURI string) {
	log := logd.FromContext(ctx)
	oldImageID := target.ImageID

	target.ImageID = imageURI
	target.Version = string(status.CustomImageVersionSource)

	log.Info("updated image version info",
		"oldImageID", oldImageID,
		"newImageID", target.ImageID)
}

func (u updater) Name() string {
	return "edgeconnect"
}

func (u updater) Target() *status.VersionStatus {
	return &u.edgeConnect.Status.Version
}

func (u updater) IsAutoUpdateEnabled() bool {
	return u.edgeConnect.IsAutoUpdateEnabled()
}

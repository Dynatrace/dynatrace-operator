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
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const minRequestThreshold = 15 * time.Minute

type updater struct {
	edgeConnect         *edgeconnect.EdgeConnect
	apiReader           client.Reader
	timeProvider        *timeprovider.Provider
	imageClientProvider ImageClientProvider
}

var _ versionStatusUpdater = updater{}

func newUpdater(
	apiReader client.Reader,
	timeprovider *timeprovider.Provider,
	imageClientProvider ImageClientProvider,
	ec *edgeconnect.EdgeConnect,
) *updater {
	return &updater{
		edgeConnect:         ec,
		apiReader:           apiReader,
		timeProvider:        timeprovider,
		imageClientProvider: imageClientProvider,
	}
}

func (u updater) RequiresReconcile() bool {
	version := u.edgeConnect.Status.Version

	if version.ImageID == "" {
		return true
	}

	if u.edgeConnect.IsCustomImage() {
		// a custom image is taken over as-is, so any change of the image field has to be applied right away
		return version.Source != status.CustomImageVersionSource ||
			!strings.HasPrefix(version.ImageID, u.edgeConnect.Image())
	}

	// switching away from a custom image, or to a different public registry, has to be applied right
	// away, otherwise the image is only refreshed from fleet management if auto update is enabled
	if version.Source != status.PublicRegistryVersionSource || u.didPublicRegistryChange(version.ImageID) {
		return true
	}

	return u.timeProvider.IsOutdated(version.LastProbeTimestamp, minRequestThreshold) && u.IsAutoUpdateEnabled()
}

// didPublicRegistryChange reports whether the image in the status was pulled from a registry other
// than the one currently requested. Without an override the registry is chosen by fleet management,
// so there is nothing to compare the status against.
func (u updater) didPublicRegistryChange(imageID string) bool {
	registryOverride := u.edgeConnect.Spec.PublicRegistryOverride
	if registryOverride == "" {
		return false
	}

	return !strings.HasPrefix(imageID, registryOverride+"/")
}

func (u updater) Update(ctx context.Context) error {
	log := logd.FromContext(ctx)

	var err error

	defer func() {
		if err == nil {
			u.Target().LastProbeTimestamp = u.timeProvider.Now()
		}
	}()

	target := u.Target()
	imageID := u.edgeConnect.Image()

	if !u.edgeConnect.IsCustomImage() {
		log.Debug("EdgeConnect public registry image used")

		var imagesClient dtimage.Client

		imagesClient, err = u.imageClientProvider(ctx)
		if err != nil {
			return err
		}

		var info *dtimage.Info

		info, err = imagesClient.GetComponentLatestInfo(ctx, dtimage.EdgeConnect, u.edgeConnect.Spec.PublicRegistryOverride)
		if err != nil {
			return err
		}

		imageID = info.URI
		target.Source = status.PublicRegistryVersionSource
	} else {
		log.Debug("EdgeConnect custom image used")

		target.Source = status.CustomImageVersionSource
	}

	target.ImageID = imageID

	return nil
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

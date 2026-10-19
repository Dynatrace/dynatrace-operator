// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"context"
	"errors"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

type ImageResolver interface {
	GetImage() string
	SetImage(imageURI string)
	PublicRegistryOverride() string
}

// ResolveImage checks if the provided resolver already has an image. If not it looks up the image using the component type.
// Retries the image lookup for every provided optional fallback or until a Dynatrace API error is encountered.
// On success, sets the image URI on the ImageResolver.
func ResolveImage(ctx context.Context, r ImageResolver, clt image.Client, component image.ComponentType, fallback ...image.ComponentType) error {
	if imageURI := r.GetImage(); imageURI != "" {
		r.SetImage(imageURI)

		return nil
	}

	imageInfo, err := clt.GetComponentLatestInfo(ctx, component, r.PublicRegistryOverride())
	if err != nil {
		if apiErr := new(core.HTTPError); len(fallback) == 0 || errors.As(err, &apiErr) {
			return err
		}

		for idx, component := range fallback {
			imageInfo, err = clt.GetComponentLatestInfo(ctx, component, r.PublicRegistryOverride())
			if err != nil {
				if apiErr := new(core.HTTPError); idx >= len(fallback)-1 || errors.As(err, &apiErr) {
					return err
				}

				continue
			}

			logd.FromContext(ctx).Debug("using fallback image name for backward compatibility", "fallback", component, "imageURI", imageInfo.URI)

			break
		}
	}

	r.SetImage(imageInfo.URI)

	return nil
}

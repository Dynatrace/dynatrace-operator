package registry

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/image"
	dtimage "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/image"
	"github.com/pkg/errors"
)

// ResolveImage resolves the image URI for a component.
// Template image takes precedence over public registry. Returns an error if neither is configured.
func ResolveImage(ctx context.Context, imageClient dtimage.Client, isPublicRegistry bool, registryOverride string, component dtimage.ComponentType, templateRef *image.Ref) (string, error) { //nolint: revive
	if templateRef.HasImage() {
		return templateRef.String(), nil
	}

	if !isPublicRegistry {
		return "", errors.Errorf("no image configured for component %q: set a template image or enable the public registry feature flag", component)
	}

	imageInfo, err := imageClient.GetComponentLatestInfo(ctx, component, registryOverride)
	if err != nil {
		return "", err
	}

	return imageInfo.URI, nil
}

package registry

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/image"
)

func ResolveImage(ctx context.Context, imageClient image.Client, isPublicRegistry bool, registryOverride string, component image.ComponentType) (string, error) {
	if !isPublicRegistry {
		return "", nil
	}

	imageInfo, err := imageClient.GetComponentLatestInfo(ctx, component, registryOverride)
	if err != nil {
		return "", err
	}

	return imageInfo.URI, nil
}

package registry

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/image"
)

func ResolveImage(ctx context.Context, imageClient image.Client, dk *dynakube.DynaKube, component image.ComponentType) (string, error) {
	if !dk.FF().IsPublicRegistry() {
		return "", nil
	}

	imageInfo, err := imageClient.GetComponentLatestInfo(ctx, component, dk.PublicRegistryOverride())
	if err != nil {
		return "", err
	}

	return imageInfo.URI, nil
}

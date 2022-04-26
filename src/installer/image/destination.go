package image

import (
	"fmt"

	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
)

func getDestinationInfo(imageCacheDir string) (*types.SystemContext, *types.ImageReference, error) {
	destinationRef, err := getDestinationReference(imageCacheDir)
	if err != nil {
		return nil, nil, err
	}
	destinationCtx := buildDestinationContext(imageCacheDir)
	return destinationCtx, &destinationRef, nil
}

func buildDestinationContext(cacheDir string) *types.SystemContext {
	return &types.SystemContext{
		BlobInfoCacheDir:   cacheDir,
		DirForceDecompress: true,
	}
}

func getDestinationReference(imageCacheDir string) (types.ImageReference, error) {
	destinationImage := fmt.Sprintf("oci:%s", imageCacheDir)

	return alltransports.ParseImageName(destinationImage)
}

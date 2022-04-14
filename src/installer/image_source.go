package installer

import (
	"github.com/Dynatrace/dynatrace-operator/src/dockerconfig"
	"github.com/containers/image/v5/docker"
	"github.com/containers/image/v5/docker/reference"
	"github.com/containers/image/v5/types"
)

func getSourceInfo(cacheDir string, pullInfo ImageInfo) (*types.SystemContext, *types.ImageReference, error) {
	imageRef, err := parseImageReference(pullInfo.Image)
	if err != nil {
		log.Info("failed to parse image reference", "image", pullInfo.Image)
		return nil, nil, err
	}
	log.Info("parsed image reference", "imageRef", imageRef)

	sourceRef, err := getSourceReference(imageRef)
	if err != nil {
		log.Info("failed to get source reference", "image", pullInfo.Image, "imageRef", imageRef)
		return nil, nil, err
	}
	log.Info("got source reference", "image", pullInfo.Image)

	sourceCtx := buildSourceContext(imageRef, cacheDir, pullInfo.DockerConfig)
	return sourceCtx, &sourceRef, nil
}

func parseImageReference(uri string) (reference.Named, error) {
	ref, err := reference.ParseNormalizedNamed(uri)
	if err != nil {
		return nil, err
	}
	ref = reference.TagNameOnly(ref)

	return ref, nil
}

func getSourceReference(named reference.Named) (types.ImageReference, error) {
	return docker.NewReference(named)
}

func buildSourceContext(imageRef reference.Named, cacheDir string, dockerConfig dockerconfig.DockerConfig) *types.SystemContext {
	systemContext := dockerconfig.MakeSystemContext(imageRef, &dockerConfig)
	systemContext.BlobInfoCacheDir = cacheDir
	return systemContext
}

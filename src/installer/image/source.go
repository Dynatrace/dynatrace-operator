package image

import (
	"github.com/Dynatrace/dynatrace-operator/src/dockerconfig"
	"github.com/containers/image/v5/docker"
	"github.com/containers/image/v5/docker/reference"
	"github.com/containers/image/v5/types"
)

func getSourceInfo(cacheDir string, pullInfo Properties) (*types.SystemContext, *types.ImageReference, error) {
	imageRef, err := parseImageReference(pullInfo.ImageUri)
	if err != nil {
		log.Info("failed to parse image reference", "image", pullInfo.ImageUri)
		return nil, nil, err
	}
	log.Info("parsed image reference", "imageRef", imageRef)

	sourceRef, err := getSourceReference(imageRef)
	if err != nil {
		log.Info("failed to get source reference", "image", pullInfo.ImageUri, "imageRef", imageRef)
		return nil, nil, err
	}
	log.Info("got source reference", "image", pullInfo.ImageUri)

	sourceCtx := buildSourceContext(imageRef, cacheDir, pullInfo.DockerConfig)
	return sourceCtx, sourceRef, nil
}

func parseImageReference(uri string) (reference.Named, error) {
	ref, err := reference.ParseNormalizedNamed(uri)
	if err != nil {
		return nil, err
	}
	ref = reference.TagNameOnly(ref)

	return ref, nil
}

func getSourceReference(named reference.Named) (*types.ImageReference, error) {
	ref, err := docker.NewReference(named)
	return &ref, err
}

func buildSourceContext(imageRef reference.Named, cacheDir string, dockerConfig dockerconfig.DockerConfig) *types.SystemContext {
	systemContext := dockerconfig.MakeSystemContext(imageRef, &dockerConfig)
	systemContext.BlobInfoCacheDir = cacheDir
	return systemContext
}

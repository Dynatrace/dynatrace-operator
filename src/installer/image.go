package installer

import (
	"context"
	"path/filepath"

	dtcsi "github.com/Dynatrace/dynatrace-operator/src/controllers/csi"
	"github.com/containers/image/v5/docker"
	"github.com/containers/image/v5/types"
	"github.com/opencontainers/go-digest"
)

var ImageCacheDir = filepath.Join(dtcsi.DataPath + "cache")

func (installer *OneAgentInstaller) installAgentFromImage(targetDir string) error {
	_ = installer.fs.MkdirAll(ImageCacheDir, 0755)
	image := installer.props.ImageInfo.Image

	sourceCtx, sourceRef, err := getSourceInfo(ImageCacheDir, *installer.props.ImageInfo)
	if err != nil {
		log.Info("failed to get source information", "image", image)
		return err
	}

	imageDigest, err := getImageDigest(sourceCtx, sourceRef)
	if err != nil {
		log.Info("failed to get image digest", "image", image)
		return err
	}
	imageCacheDir := filepath.Join(ImageCacheDir, imageDigest.Encoded())

	destinationCtx, destinationRef, err := getDestinationInfo(imageCacheDir)
	if err != nil {
		log.Info("failed to get destination information", "image", image, "imageCacheDir", imageCacheDir)
		return err
	}
	return installer.getAgentFromImage(
		imagePullInfo{
			imageCacheDir:  imageCacheDir,
			targetDir:      targetDir,
			sourceCtx:      sourceCtx,
			destinationCtx: destinationCtx,
			sourceRef:      sourceRef,
			destinationRef: destinationRef,
		},
	)

}

func getImageDigest(systemContext *types.SystemContext, imageReference *types.ImageReference) (digest.Digest, error) {
	return docker.GetDigest(context.TODO(), systemContext, *imageReference)
}

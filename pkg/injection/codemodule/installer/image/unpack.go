package image

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"

	"github.com/Dynatrace/dynatrace-operator/pkg/arch"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/common"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	containerv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/pkg/errors"
)

type imagePullInfo struct {
	imageCacheDir string
	targetDir     string
}

func (installer *Installer) extractAgentBinariesFromImage(pullInfo imagePullInfo, imageName string) error { //nolint
	img, err := installer.pullImageInfo(imageName)
	if err != nil {
		log.Info("pullImageInfo", "error", err)

		return err
	}

	image := *img

	err = installer.pullOCIimage(image, imageName, pullInfo.imageCacheDir, pullInfo.targetDir)
	if err != nil {
		log.Info("pullOCIimage", "err", err)

		return err
	}

	return nil
}

func (installer *Installer) pullImageInfo(imageName string) (*containerv1.Image, error) {
	ref, err := name.ParseReference(imageName)
	if err != nil {
		return nil, errors.WithMessagef(err, "parsing reference %q:", imageName)
	}

	image, err := remote.Image(ref, remote.WithContext(context.TODO()),
		remote.WithAuthFromKeychain(installer.keychain),
		remote.WithTransport(installer.transport),
		remote.WithPlatform(arch.ImagePlatform),
	)
	if err != nil {
		return nil, errors.WithMessagef(err, "getting image %q", imageName)
	}

	return &image, nil
}

func (installer *Installer) pullOCIimage(image containerv1.Image, imageName string, imageCacheDir string, targetDir string) error {
	ref, err := name.ParseReference(imageName)
	if err != nil {
		return errors.WithMessagef(err, "parsing reference %q", imageName)
	}

	log.Info("pullOciImage", "ref_identifier", ref.Identifier(), "ref.Name", ref.Name(), "ref.String", ref.String())

	err = os.MkdirAll(imageCacheDir, common.MkDirFileMode)
	if err != nil {
		log.Info("failed to create cache dir", "dir", imageCacheDir, "err", err)

		return errors.WithStack(err)
	}

	// ref.String() is consistent with what the user gave, ref.Name() could add some prefix depending on the situation.
	// It doesn't really matter here as it's only for a temporary dir, but it's still better be consistent.
	imageCachePath := filepath.Join(imageCacheDir, base64.StdEncoding.EncodeToString([]byte(ref.String())))
	if err := crane.SaveOCI(image, imageCachePath); err != nil {
		log.Info("saving v1.Image img as an OCI Image Layout at path", imageCacheDir, err)

		return errors.WithMessagef(err, "saving v1.Image img as an OCI Image Layout at path %s", imageCacheDir)
	}

	layers, err := image.Layers()
	if err != nil {
		log.Info("failed to get image layers", "err", err)

		return errors.WithStack(err)
	}

	err = installer.unpackOciImage(layers, imageCachePath, targetDir)
	if err != nil {
		log.Info("failed to unpackOciImage", "error", err)

		return errors.WithStack(err)
	}

	return nil
}

func (installer *Installer) unpackOciImage(layers []containerv1.Layer, imageCacheDir string, targetDir string) error {
	for _, layer := range layers {
		mediaType, _ := layer.MediaType()
		switch mediaType {
		case types.DockerLayer, types.OCILayer:
			digest, _ := layer.Digest()
			sourcePath := filepath.Join(imageCacheDir, "blobs", digest.Algorithm, digest.Hex)
			log.Info("unpackOciImage", "sourcePath", sourcePath)

			if err := installer.extractor.ExtractGzip(sourcePath, targetDir); err != nil {
				return err
			}
		case types.OCILayerZStd:
			return errors.New("OCILayerZStd is not implemented")
		default:
			return errors.Errorf("media type %s is not implemented", mediaType)
		}
	}

	log.Info("unpackOciImage", "targetDir", targetDir)

	return nil
}

package image

import (
	"context"
	"encoding/base64"
	goerrors "errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Dynatrace/dynatrace-operator/pkg/arch"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/common"
	"github.com/containerd/containerd/archive"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	containerv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/partial"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/pkg/errors"
)

type imagePullInfo struct {
	imageCacheDir string
	targetDir     string
}

func (installer *Installer) extractAgentBinariesFromImage(pullInfo imagePullInfo, imageName string) error {
	ctx := context.TODO()

	ref, err := name.ParseReference(imageName)
	if err != nil {
		return errors.WithMessagef(err, "parse reference %s", imageName)
	}

	image, err := remote.Image(ref, remote.WithContext(ctx),
		remote.WithAuthFromKeychain(installer.keychain),
		remote.WithTransport(installer.transport),
		remote.WithPlatform(arch.ImagePlatform),
	)
	if err != nil {
		return errors.WithMessagef(err, "get image %s", imageName)
	}

	unpackDir := filepath.Join(pullInfo.imageCacheDir, base64.StdEncoding.EncodeToString([]byte(ref.String())))
	if err := unpackImage(ctx, image, unpackDir); err != nil {
		return errors.WithMessagef(err, "unpack image %s", imageName)
	}

	log.Info("moving unpacked archive to target", "targetDir", pullInfo.targetDir)

	if err := os.Rename(filepath.Join(unpackDir, "opt", "dynatrace", "oneagent"), pullInfo.targetDir); err != nil {
		if !os.IsNotExist(err) {
			return errors.WithMessagef(err, "move unpacked image %s", imageName)
		}

		return errors.WithMessagef(os.Rename(unpackDir, pullInfo.targetDir), "move unpacked image %s", imageName)
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

// Unpack remote OCI image into targetDir
func unpackImage(ctx context.Context, remoteImage containerv1.Image, targetDir string) error {
	if err := os.MkdirAll(targetDir, common.MkDirFileMode); err != nil {
		return errors.WithMessage(err, "create target directory")
	}

	remoteLayers, err := remoteImage.Layers()
	if err != nil {
		return errors.WithMessage(err, "get image layers")
	}

	// Could append into the remote image instead, but using this empty image requires less work when merging.
	localImage, _ := partial.UncompressedToImage(emptyImage{})

	for _, layer := range remoteLayers {
		mediaType, _ := layer.MediaType()
		if mediaType != types.DockerLayer && mediaType != types.OCILayer {
			return errors.Errorf("media type %s is not implemented", mediaType)
		}

		digest, _ := layer.Digest()
		size, _ := layer.Size()
		log.Debug("downloading layer", "size", size, "digest", digest.Hex)

		downloaded, err := tarball.LayerFromOpener(layer.Compressed)
		if err != nil {
			return errors.WithMessage(err, "create tar reader")
		}

		// Alternatively, read the compressed data into a file and use the path instead.
		// remote, err := layer.Compressed()
		// local, err := os.Create("")
		// io.Copy(local, remote)
		// tarball.LayerFromFile(local)

		localImage, err = mutate.Append(localImage, mutate.Addendum{Layer: downloaded})
		if err != nil {
			return err
		}
	}

	// This reader can also be used with tar.NewReader directly (no gzip needed)
	xr := mutate.Extract(localImage)
	_, err = archive.Apply(ctx, targetDir, xr, archive.WithNoSameOwner())

	return goerrors.Join(err, xr.Close())
}

// Copied from github.com/google/go-containerregistry@v0.20.7/pkg/v1/empty/image.go
// Because the exported variable is mutable and should not be reused.
type emptyImage struct{}

// MediaType implements partial.UncompressedImageCore.
func (i emptyImage) MediaType() (types.MediaType, error) {
	return types.DockerManifestSchema2, nil
}

// RawConfigFile implements partial.UncompressedImageCore.
func (i emptyImage) RawConfigFile() ([]byte, error) {
	return partial.RawConfigFile(i)
}

// ConfigFile implements v1.Image.
func (i emptyImage) ConfigFile() (*containerv1.ConfigFile, error) {
	return &containerv1.ConfigFile{
		RootFS: containerv1.RootFS{
			// Some clients check this.
			Type: "layers",
		},
	}, nil
}

func (i emptyImage) LayerByDiffID(h containerv1.Hash) (partial.UncompressedLayer, error) {
	return nil, fmt.Errorf("LayerByDiffID(%s): empty image", h)
}

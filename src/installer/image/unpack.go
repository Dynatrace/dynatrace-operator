package image

import (
	"context"
	"fmt"
	"path"
	"path/filepath"

	"github.com/Dynatrace/dynatrace-operator/src/dockerconfig"
	"github.com/Dynatrace/dynatrace-operator/src/dockerkeychain"
	"github.com/Dynatrace/dynatrace-operator/src/installer/common"
	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/signature"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

const (
	// MediaTypeImageLayerGzip is the media type used for gzipped layers
	// referenced by the manifest.
	mediaTypeImageLayerGzip = "application/vnd.oci.image.layer.v1.tar+gzip"

	// MediaTypeImageLayerZstd is the media type used for zstd compressed
	// layers referenced by the manifest.
	mediaTypeImageLayerZstd = "application/vnd.oci.image.layer.v1.tar+zstd"

	mediaTypeImageLayerDockerRootFs = "application/vnd.docker.image.rootfs.diff.tar.gzip"
)

type imagePullInfo struct {
	imageCacheDir string
	targetDir     string
}

func (installer Installer) extractAgentBinariesFromImage(pullInfo imagePullInfo, dockerConfig *dockerconfig.DockerConfig, imageName string) error { //nolint
	img, err := installer.pullImageInfo(dockerConfig, imageName)
	if err != nil {
		log.Info("pullImageInfo", "error", err)
		return err
	}

	image := *img

	manifest, err := image.Manifest()
	if err != nil {
		log.Info("manifest", "error", err)
		return err
	}

	if manifest.MediaType.IsIndex() {
		log.Info("manifest is index")
	}
	if manifest.MediaType.IsImage() {
		log.Info("manifest is image")
	}
	log.Info("manifest", "MediaType", manifest.MediaType)

	for _, layer := range manifest.Layers {
		log.Info("layers", "digest", layer.Digest.Hex, "type", layer.MediaType)
	}

	err = installer.pullOCIimage(image, imageName, manifest, pullInfo.imageCacheDir, pullInfo.targetDir)
	if err != nil {
		log.Info("pullOCIimage", "err", err)
		return err
	}

	return nil
}

func (installer Installer) pullImageInfo(dockerConfig *dockerconfig.DockerConfig, imageName string) (*v1.Image, error) {
	ref, err := name.ParseReference(imageName)
	if err != nil {
		return nil, fmt.Errorf("parsing reference %q: %w", imageName, err)
	}

	log.Info("ref", "refName", ref.Name(), "refString", ref.String(), "refIdentifier", ref.Identifier(), "Context().RegistryStr()", ref.Context().RegistryStr(), "Context().Name()", ref.Context().Name(), "Context().Scheme()", ref.Context().Scheme())

	keyChain := dockerkeychain.NewDockerKeychain(dockerConfig.RegistryAuthPath, installer.fs)

	image, err := remote.Image(ref, remote.WithContext(context.TODO()), remote.WithAuthFromKeychain(keyChain), remote.WithTransport(installer.httpClient.Transport), remote.WithUserAgent("ao"))
	if err != nil {
		return nil, fmt.Errorf("getting image %q: %w", imageName, err)
	}
	return &image, nil
}

func (installer Installer) pullOCIimage(image v1.Image, imageName string, _ *v1.Manifest, imageCacheDir string, targetDir string) error {
	log.Info("pullOciImage")

	ref, err := name.ParseReference(imageName)
	if err != nil {
		return fmt.Errorf("parsing reference %q: %w", imageName, err)
	}

	log.Info("pullOciImage", "ref_identifier", ref.Identifier(), "ref.Name", ref.Name(), "ref.String", ref.String())

	err = installer.fs.MkdirAll(imageCacheDir, common.MkDirFileMode)
	if err != nil {
		log.Info("failed to create cache dir", "dir", imageCacheDir, "err", err)
		return errors.WithStack(err)
	}

	if err := crane.SaveOCI(image, path.Join(imageCacheDir, ref.Identifier())); err != nil {
		log.Info("saving tarball", imageCacheDir, err)
		return fmt.Errorf("saving tarball %s: %w", imageCacheDir, err)
	}

	/*
		cacheDir/ref/index.json
		{
		   "schemaVersion": 2,
		   "mediaType": "application/vnd.oci.image.index.v1+json",
		   "manifests": [
			  {
				 "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
				 "size": 530,
				 "digest": "sha256:7ece13a07a20c77a31cc36906a10ebc90bd47970905ee61e8ed491b7f4c5d62f"
			  }
		   ]
		}

		cacheDir/ref/blobs/sha256/7ece13a07a20c77a31cc36906a10ebc90bd47970905ee61e8ed491b7f4c5d62f
		{
		   "schemaVersion": 2,
		   "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
		   "config": {
		      "mediaType": "application/vnd.docker.container.image.v1+json",
		      "size": 1177,
		      "digest": "sha256:8230a0268e11c04ab875d426c35e81f7654482e2bd5901fdb7eda90bd35469df"
		   },
		   "layers": [
		      {
		         "mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
		         "size": 282834356,
		         "digest": "sha256:e7f3c606f5adf01f0480a96f74e78074b343f0760eca065b4cf7d46a9472ac37"
		      }
		   ]
		}

		cacheDir/ref/blobs/sha256/e7f3c606f5adf01f0480a96f74e78074b343f0760eca065b4cf7d46a9472ac37
	*/

	aferoFs := afero.Afero{
		Fs: installer.fs,
	}

	manifestFile, err := aferoFs.ReadFile(filepath.Join(imageCacheDir, ref.Identifier(), "index.json"))
	if err != nil {
		log.Info("failed to read index.json", "error", err)
		return errors.WithStack(err)
	}

	manifests, err := unmarshallImageIndex(aferoFs, filepath.Join(imageCacheDir, ref.Identifier()), manifestFile)
	if err != nil {
		log.Info("failed to unmarshal manifests",
			"image", installer.props.ImageUri,
			"manifestBlob", manifestFile,
			"imageCacheDir", imageCacheDir,
		)
		return errors.WithStack(err)
	}

	err = installer.unpackOciImage(manifests, filepath.Join(imageCacheDir, ref.Identifier()), targetDir)
	if err != nil {
		log.Info("failed to unpackOciImage", "error", err)
		return errors.WithStack(err)
	}
	return nil
}

func (installer Installer) unpackOciImage(manifests []*manifest.OCI1, imageCacheDir string, targetDir string) error {
	for _, entry := range manifests {
		for _, layer := range entry.LayerInfos() {
			switch layer.MediaType {
			case mediaTypeImageLayerDockerRootFs:
				sourcePath := filepath.Join(imageCacheDir, "blobs", layer.Digest.Algorithm().String(), layer.Digest.Hex())
				log.Info("unpackOciImage", "sourcePath", sourcePath)
				if err := installer.extractor.ExtractGzip(sourcePath, targetDir); err != nil {
					return err
				}
			case mediaTypeImageLayerGzip:
				return fmt.Errorf("MediaTypeImageLayerGzip is not implemented")
			case mediaTypeImageLayerZstd:
				return fmt.Errorf("MediaTypeImageLayerZstd is not implemented")

			default:
				return fmt.Errorf("unknown media type: %s", layer.MediaType)
			}
		}
	}
	log.Info("unpackOciImage", "targetDir", targetDir)
	return nil
}

func unmarshallImageIndex(fs afero.Fs, imageCacheDir string, manifestBlob []byte) ([]*manifest.OCI1, error) {
	index, err := manifest.OCI1IndexFromManifest(manifestBlob)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	aferoFs := afero.Afero{
		Fs: fs,
	}

	var manifests []*manifest.OCI1 // nolint:prealloc
	for _, descriptor := range index.Manifests {
		manifestFile, err := aferoFs.ReadFile(filepath.Join(imageCacheDir, "blobs", descriptor.Digest.Algorithm().String(), descriptor.Digest.Hex()))
		if err != nil {
			return nil, errors.WithStack(err)
		}
		ociManifest, err := manifest.OCI1FromManifest(manifestFile)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		manifests = append(manifests, ociManifest)
	}
	return manifests, nil
}

func buildPolicyContext() (*signature.PolicyContext, error) {
	policy, err := signature.NewPolicyFromBytes([]byte(rawPolicy))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return signature.NewPolicyContext(policy)
}

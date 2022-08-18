package image

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/types"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

type imagePullInfo struct {
	imageCacheDir  string
	targetDir      string
	sourceCtx      *types.SystemContext
	destinationCtx *types.SystemContext
	sourceRef      *types.ImageReference
	destinationRef *types.ImageReference
}

func (installer ImageInstaller) extractAgentBinariesFromImage(pullInfo imagePullInfo) error {
	manifestBlob, err := copyImageToCache(pullInfo)
	if err != nil {
		log.Info("failed to get manifests blob",
			"image", installer.props.ImageUri,
		)
		return errors.WithStack(err)
	}

	manifests, err := installer.unmarshalManifestBlob(manifestBlob, pullInfo.imageCacheDir)
	if err != nil {
		log.Info("failed to unmarshal manifests",
			"image", installer.props.ImageUri,
			"manifestBlob", manifestBlob,
			"imageCacheDir", pullInfo.imageCacheDir,
		)
		return errors.WithStack(err)
	}
	return installer.unpackOciImage(manifests, pullInfo.imageCacheDir, pullInfo.targetDir)

}

func (installer ImageInstaller) unmarshalManifestBlob(manifestBlob []byte, imageCacheDir string) ([]*manifest.OCI1, error) {
	var manifests []*manifest.OCI1

	switch manifest.GuessMIMEType(manifestBlob) {
	case ocispec.MediaTypeImageManifest:
		ociManifest, err := manifest.OCI1FromManifest(manifestBlob)
		if err != nil {
			return manifests, errors.WithStack(err)
		}
		manifests = append(manifests, ociManifest)
	case ocispec.MediaTypeImageIndex:
		ociManifests, err := unmarshallImageIndex(installer.fs, imageCacheDir, manifestBlob)
		if err != nil {
			return manifests, errors.WithStack(err)
		}
		manifests = append(manifests, ociManifests...)
	}

	return manifests, nil
}

func (installer ImageInstaller) unpackOciImage(manifests []*manifest.OCI1, imageCacheDir string, targetDir string) error {
	for _, entry := range manifests {
		for _, layer := range entry.LayerInfos() {
			switch layer.MediaType {
			case ocispec.MediaTypeImageLayerGzip:
				sourcePath := filepath.Join(imageCacheDir, "blobs", layer.Digest.Algorithm().String(), layer.Digest.Hex())
				if err := installer.extractor.ExtractGzip(sourcePath, targetDir); err != nil {
					return err
				}
			case ocispec.MediaTypeImageLayerZstd:
				return fmt.Errorf("MediaTypeImageLayerZstd is not implemented")
			default:
				return fmt.Errorf("unknown media type: %s", layer.MediaType)
			}
		}
	}
	return nil
}

func unmarshallImageIndex(fs afero.Fs, imageCacheDir string, manifestBlob []byte) ([]*manifest.OCI1, error) {
	var manifests []*manifest.OCI1
	index, err := manifest.OCI1IndexFromManifest(manifestBlob)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	aferoFs := afero.Afero{
		Fs: fs,
	}
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

func copyImageToCache(pullInfo imagePullInfo) ([]byte, error) {
	policyCtx, err := buildPolicyContext()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer func() { _ = policyCtx.Destroy() }()

	manifestBlob, err := copy.Image(context.TODO(), policyCtx, *pullInfo.destinationRef, *pullInfo.sourceRef, &copy.Options{
		SourceCtx:                             pullInfo.sourceCtx,
		DestinationCtx:                        pullInfo.destinationCtx,
		OptimizeDestinationImageAlreadyExists: true,
	})

	return manifestBlob, err
}

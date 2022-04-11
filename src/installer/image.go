package installer

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/src/dockerconfig"
	"github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/docker"
	"github.com/containers/image/v5/docker/reference"
	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
	"github.com/klauspost/compress/gzip"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

func (installer *OneAgentInstaller) installAgentFromImage(targetDir string) error {
	cacheDir := filepath.Join(targetDir, "cache")
	_ = installer.fs.MkdirAll(cacheDir, 0755)
	defer func() { installer.fs.RemoveAll(cacheDir) }()
	image := installer.props.ImageInfo.Image

	imageRef, err := parseImageReference(image)
	if err != nil {
		log.Info("failed to parse image reference", "image", image)
		return err
	}
	sourceRef, err := getSourceReference(imageRef)
	if err != nil {
		log.Info("failed to get source reference", "image", image, "imageRef", imageRef)
		return err
	}

	sourceCtx := buildSourceContext(imageRef, cacheDir, installer.props.ImageInfo.DockerConfig)
	destinationCtx := buildDestinationContext(cacheDir)

	digest, err := getImageDigest(sourceCtx, sourceRef)
	if err != nil {
		log.Info("failed to get image digest", "image", image, "sourceRef", sourceRef, "sourceCtx", sourceCtx)
		return err
	}

	imageCacheDir := filepath.Join(cacheDir, digest.Encoded())
	destinationRef, err := getDestinationReference(imageCacheDir)
	if err != nil {
		log.Info("failed to get destination ref", "image", image, "imageTargetDir", imageCacheDir)
		return err
	}

	manifestBlob, err := copyImageToCache(sourceCtx, destinationCtx, sourceRef, destinationRef)
	if err != nil {
		log.Info("failed to get manifests blob",
			"image", image,
			"sourceCtx", sourceCtx,
			"destinationCtx", destinationCtx,
			"sourceRef", sourceRef,
			"destinationRef", destinationRef,
		)
		return err
	}

	manifests, err := unmarshalManifestBlob(manifestBlob, imageCacheDir)
	if err != nil {
		log.Info("failed to unmarshal manifests",
			"image", image,
			"manifestBlob", manifestBlob,
			"imageCacheDir", imageCacheDir,
		)
		return err
	}

	return unpackOciImage(manifests, imageCacheDir, targetDir)
}

func parseImageReference(uri string) (reference.Named, error) {
	ref, err := reference.ParseNormalizedNamed(uri)
	if err != nil {
		return nil, err
	}
	ref = reference.TagNameOnly(ref)

	return ref, nil
}

func buildSourceContext(imageRef reference.Named, cacheDir string, dockerConfig dockerconfig.DockerConfig) *types.SystemContext {
	systemContext := dockerconfig.MakeSystemContext(imageRef, &dockerConfig)
	systemContext.BlobInfoCacheDir = cacheDir
	return systemContext
}

func buildDestinationContext(cacheDir string) *types.SystemContext {
	return &types.SystemContext{
		BlobInfoCacheDir:   cacheDir,
		DirForceDecompress: true,
	}
}

func getImageDigest(systemContext *types.SystemContext, imageReference types.ImageReference) (digest.Digest, error) {
	return docker.GetDigest(context.TODO(), systemContext, imageReference)
}

func getSourceReference(named reference.Named) (types.ImageReference, error) {
	return docker.NewReference(named)
}

func getDestinationReference(imageCacheDir string) (types.ImageReference, error) {
	destinationImage := fmt.Sprintf("oci:%s", imageCacheDir)

	return alltransports.ParseImageName(destinationImage)
}

func buildPolicyContext(sourceContext *types.SystemContext) (*signature.PolicyContext, error) {
	policy, err := signature.DefaultPolicy(sourceContext)
	if err != nil {
		return nil, err
	}

	return signature.NewPolicyContext(policy)
}

func copyImageToCache(sourceCtx *types.SystemContext, destinationCtx *types.SystemContext, sourceRef types.ImageReference, destinationRef types.ImageReference) ([]byte, error) {
	policyCtx, err := buildPolicyContext(sourceCtx)
	if err != nil {
		return nil, nil
	}
	defer func() { _ = policyCtx.Destroy() }()

	manifestBlob, err := copy.Image(context.TODO(), policyCtx, destinationRef, sourceRef, &copy.Options{
		SourceCtx:                             sourceCtx,
		DestinationCtx:                        destinationCtx,
		OptimizeDestinationImageAlreadyExists: true,
	})

	return manifestBlob, err
}

func unmarshalManifestBlob(manifestBlob []byte, imageCacheDir string) ([]*manifest.OCI1, error) {
	var manifests []*manifest.OCI1

	mimeType := manifest.GuessMIMEType(manifestBlob)

	if mimeType == ocispec.MediaTypeImageManifest {
		ociManifest, err := manifest.OCI1FromManifest(manifestBlob)
		if err != nil {
			return manifests, err
		}
		manifests = append(manifests, ociManifest)

	} else if mimeType == ocispec.MediaTypeImageIndex {
		index, err := manifest.OCI1IndexFromManifest(manifestBlob)
		if err != nil {
			return nil, err
		}
		for _, descriptor := range index.Manifests {
			mBlob, err := os.ReadFile(filepath.Join(imageCacheDir, "blobs", descriptor.Digest.Algorithm().String(), descriptor.Digest.Hex()))
			if err != nil {
				return nil, err
			}
			ociManifest, err := manifest.OCI1FromManifest(mBlob)
			if err != nil {
				return nil, err
			}
			manifests = append(manifests, ociManifest)
		}
	}

	return manifests, nil
}

func unpackOciImage(manifests []*manifest.OCI1, imageCacheDir string, destination string) error {
	for _, entry := range manifests {
		for _, layer := range entry.LayerInfos() {
			switch layer.MediaType {
			case ocispec.MediaTypeImageLayerGzip:
				source := filepath.Join(imageCacheDir, "blobs", layer.Digest.Algorithm().String(), layer.Digest.Hex())
				if err := extractTarGzip(source, destination); err != nil {
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

func extractTarGzip(source string, destinationDir string) error {
	destinationDir = filepath.Clean(destinationDir)

	reader, err := os.Open(source)
	if err != nil {
		return err
	}
	defer func() { _ = reader.Close() }()

	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		return err
	}
	defer func() { _ = gzipReader.Close() }()

	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}

		target := filepath.Join(destinationDir, header.Name)

		// Check for ZipSlip: https://snyk.io/research/zip-slip-vulnerability
		if !strings.HasPrefix(target, destinationDir) {
			return fmt.Errorf("illegal file path: %s", target)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}

		case tar.TypeLink:
			if err := os.Link(filepath.Join(destinationDir, header.Linkname), target); err != nil {
				return err
			}

		case tar.TypeSymlink:
			if err := os.Symlink(header.Linkname, target); err != nil {
				return err
			}

		case tar.TypeReg:
			destinationFile, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			if _, err := io.Copy(destinationFile, tarReader); err != nil {
				return err
			}
			_ = destinationFile.Close()

		default:
			fmt.Printf("skipping special file: %s\n", header.Name)
		}
	}
}

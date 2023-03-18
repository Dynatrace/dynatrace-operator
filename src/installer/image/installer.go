package image

import (
	"context"
	"github.com/containers/image/v5/docker/reference"
	"os"
	"path/filepath"

	"github.com/Dynatrace/dynatrace-operator/src/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/src/dockerconfig"
	dtypes "github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/installer/common"
	"github.com/Dynatrace/dynatrace-operator/src/installer/symlink"
	"github.com/Dynatrace/dynatrace-operator/src/installer/zip"
	"github.com/Dynatrace/dynatrace-operator/src/processmoduleconfig"
	"github.com/containers/image/v5/docker"
	"github.com/containers/image/v5/types"
	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

type Properties struct {
	ImageUri     string
	DockerConfig dockerconfig.DockerConfig
	PathResolver metadata.PathResolver
	Metadata     metadata.Access
	imageDigest  string
}

func NewImageInstaller(fs afero.Fs, props *Properties) *ImageInstaller {
	return &ImageInstaller{
		fs:        fs,
		extractor: zip.NewOneAgentExtractor(fs, props.PathResolver),
		props:     props,
	}
}

type ImageInstaller struct {
	fs        afero.Fs
	extractor zip.Extractor
	props     *Properties
}

func (installer ImageInstaller) ImageDigest() string {
	return installer.props.imageDigest
}

func (installer *ImageInstaller) InstallAgent(targetDir string) (bool, error) {
	log.Info("installing agent from image")

	err := installer.fs.MkdirAll(installer.props.PathResolver.AgentSharedBinaryDirBase(), common.MkDirFileMode)
	if err != nil {
		log.Info("failed to create the base shared agent directory", "err", err)
		return false, errors.WithStack(err)
	}

	if err := installer.installAgentFromImage(); err != nil {
		_ = installer.fs.RemoveAll(targetDir)
		log.Info("failed to install agent from image", "err", err)
		return false, errors.WithStack(err)
	}

	sharedDir := installer.props.PathResolver.AgentSharedBinaryDirForImage(installer.ImageDigest())
	if err := symlink.CreateSymlinkForCurrentVersionIfNotExists(installer.fs, sharedDir); err != nil {
		_ = installer.fs.RemoveAll(targetDir)
		_ = installer.fs.RemoveAll(sharedDir)
		log.Info("failed to create symlink for agent installation", "err", err)
		return false, errors.WithStack(err)
	}
	return true, nil
}

func (installer ImageInstaller) UpdateProcessModuleConfig(targetDir string, processModuleConfig *dtypes.ProcessModuleConfig) error {
	sourceDir := installer.props.PathResolver.AgentSharedBinaryDirForImage(installer.ImageDigest())
	return processmoduleconfig.CreateAgentConfigDir(installer.fs, targetDir, sourceDir, processModuleConfig)
}

func (installer *ImageInstaller) installAgentFromImage() error {
	defer installer.fs.RemoveAll(CacheDir)
	err := installer.fs.MkdirAll(CacheDir, common.MkDirFileMode)
	if err != nil {
		log.Info("failed to create cache dir", "err", err)
		return errors.WithStack(err)
	}
	image := installer.props.ImageUri

	sourceCtx, sourceRef, err := getSourceInfo(CacheDir, *installer.props)
	if err != nil {
		log.Info("failed to get source information", "image", image)
		return errors.WithStack(err)
	}

	imageDigestEncoded, err := getImageDigestEncoded(sourceCtx, sourceRef, image)
	if err != nil {
		log.Info("failed to get image digest", "image", image)
		return errors.WithStack(err)
	}

	if installer.isAlreadyDownloaded(imageDigestEncoded) {
		log.Info("image is already installed", "image", image, "digest", imageDigestEncoded)
		installer.props.imageDigest = imageDigestEncoded
		return nil
	}

	imageCacheDir := getCacheDirPath(imageDigestEncoded)
	destinationCtx, destinationRef, err := getDestinationInfo(imageCacheDir)
	if err != nil {
		log.Info("failed to get destination information", "image", image, "imageCacheDir", imageCacheDir)
		return errors.WithStack(err)
	}

	err = installer.extractAgentBinariesFromImage(
		imagePullInfo{
			imageCacheDir:  imageCacheDir,
			targetDir:      installer.props.PathResolver.AgentSharedBinaryDirForImage(imageDigestEncoded),
			sourceCtx:      sourceCtx,
			destinationCtx: destinationCtx,
			sourceRef:      sourceRef,
			destinationRef: destinationRef,
		},
	)
	if err != nil {
		log.Info("failed to extract agent binaries from image", "image", image, "imageCacheDir", imageCacheDir)
		return errors.WithStack(err)
	}
	installer.props.imageDigest = imageDigestEncoded

	return nil
}

func (installer ImageInstaller) isAlreadyDownloaded(imageDigestEncoded string) bool {
	sharedDir := installer.props.PathResolver.AgentSharedBinaryDirForImage(imageDigestEncoded)
	_, err := installer.fs.Stat(sharedDir)
	return !os.IsNotExist(err)
}

func getImageDigestEncoded(systemContext *types.SystemContext, imageReference *types.ImageReference, imageName string) (string, error){
	if digestFromName := getImageDigestFromImageName(imageName); digestFromName != ""{
		log.Info("Image digest found in Image name", "imageName", imageName)
		return digestFromName.Encoded(), nil
	}

	digestFromRef, err := getImageDigestFromReference(systemContext, imageReference)
	if err != nil {
		return "", err
	}
	return digestFromRef.Encoded(), nil
}

func getImageDigestFromReference(systemContext *types.SystemContext, imageReference *types.ImageReference) (digest.Digest, error) {
	return docker.GetDigest(context.TODO(), systemContext, *imageReference)
}

func getImageDigestFromImageName(name string) digest.Digest{
	ref, err := reference.Parse(name)
	if err != nil{
		return ""
	}

	if _, ok:= ref.(reference.Canonical); ok{
		return ref.(reference.Canonical).Digest()
	}
	return ""
}

func getCacheDirPath(digest string) string {
	return filepath.Join(CacheDir, digest)
}

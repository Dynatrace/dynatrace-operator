package image

import (
	"context"
	"os"
	"path/filepath"

	"github.com/Dynatrace/dynatrace-operator/src/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/src/dockerconfig"
	dtypes "github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/installer/common"
	"github.com/Dynatrace/dynatrace-operator/src/installer/symlink"
	"github.com/Dynatrace/dynatrace-operator/src/processmoduleconfig"
	"github.com/containers/image/v5/docker"
	"github.com/containers/image/v5/types"
	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

type Properties struct {
	ImageUri     string
	ImageDigest  string
	DockerConfig dockerconfig.DockerConfig
	PathResolver metadata.PathResolver
}

func NewImageInstaller(fs afero.Fs, props *Properties) *ImageInstaller {
	return &ImageInstaller{
		fs:    fs,
		props: props,
	}
}

type ImageInstaller struct {
	fs    afero.Fs
	props *Properties
}

func (installer ImageInstaller) ImageDigest() string {
	return installer.props.ImageDigest
}

func (installer *ImageInstaller) InstallAgent(targetDir string) error {
	log.Info("installing agent from image")
	err := installer.fs.MkdirAll(targetDir, common.MkDirFileMode)
	if err != nil {
		log.Info("failed to create install target dir", "err", err, "targetDir", targetDir)
		return errors.WithStack(err)
	}
	if err := installer.installAgentFromImage(); err != nil {
		_ = installer.fs.RemoveAll(targetDir)
		log.Info("failed to install agent from image", "err", err)
		return errors.WithStack(err)
	}
	sharedDir := installer.props.PathResolver.AgentSharedBinaryDirForImage(installer.props.ImageDigest)
	return symlink.CreateSymlinkForCurrentVersionIfNotExists(installer.fs, sharedDir)
}

func (installer ImageInstaller) UpdateProcessModuleConfig(targetDir string, processModuleConfig *dtypes.ProcessModuleConfig) error {
	sourceDir := installer.props.PathResolver.AgentSharedBinaryDirForImage(installer.props.ImageDigest)
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

	imageDigest, err := getImageDigest(sourceCtx, sourceRef)
	if err != nil {
		log.Info("failed to get image digest", "image", image)
		return errors.WithStack(err)
	}

	imageDigestEncoded := imageDigest.Encoded()
	imageCacheDir := getCacheDirPath(imageDigestEncoded)
	if installer.isAlreadyDownloaded(imageDigestEncoded, imageCacheDir) {
		log.Info("image is already installed", "image", image, "digest", imageDigestEncoded)
		installer.props.ImageDigest = imageDigestEncoded
		return nil
	}
	sharedDir := installer.props.PathResolver.AgentSharedBinaryDirForImage(imageDigestEncoded)
	err = installer.fs.MkdirAll(sharedDir, common.MkDirFileMode)
	if err != nil {
		log.Info("failed to create share dir", "err", err, "sharedDir", sharedDir)
		return errors.WithStack(err)
	}

	destinationCtx, destinationRef, err := getDestinationInfo(imageCacheDir)
	if err != nil {
		log.Info("failed to get destination information", "image", image, "imageCacheDir", imageCacheDir)
		return errors.WithStack(err)
	}
	err = installer.extractAgentBinariesFromImage(
		imagePullInfo{
			imageCacheDir:  imageCacheDir,
			targetDir:      sharedDir,
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
	installer.props.ImageDigest = imageDigestEncoded
	return nil
}

func (installer ImageInstaller) isAlreadyDownloaded(imageDigestEncoded string, imageCacheDir string) bool {
	if _, err := installer.fs.Stat(imageCacheDir); !os.IsNotExist(err) {
		return false
	}
	if installer.props.ImageDigest == imageDigestEncoded {
		return true
	}
	return false
}

func getImageDigest(systemContext *types.SystemContext, imageReference *types.ImageReference) (digest.Digest, error) {
	return docker.GetDigest(context.TODO(), systemContext, *imageReference)
}

func getCacheDirPath(digest string) string {
	return filepath.Join(CacheDir, digest)
}

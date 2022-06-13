package image

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Dynatrace/dynatrace-operator/src/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/src/dockerconfig"
	dtypes "github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/installer/symlink"
	"github.com/Dynatrace/dynatrace-operator/src/processmoduleconfig"
	"github.com/containers/image/v5/docker"
	"github.com/containers/image/v5/types"
	"github.com/opencontainers/go-digest"
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

func (installer *ImageInstaller) ImageDigest() string {
	return installer.props.ImageDigest
}

func (installer *ImageInstaller) InstallAgent(targetDir string) error {
	log.Info("installing agent from image")
	_ = installer.fs.MkdirAll(targetDir, 0755)
	if err := installer.installAgentFromImage(); err != nil {
		_ = installer.fs.RemoveAll(targetDir)
		return fmt.Errorf("failed to install agent: %w", err)
	}
	sharedDir := installer.props.PathResolver.AgentSharedBinaryDirForDigest(installer.props.ImageDigest)
	return symlink.CreateSymlinkForCurrentVersionIfNotExists(installer.fs, sharedDir)
}

func (installer *ImageInstaller) UpdateProcessModuleConfig(targetDir string, processModuleConfig *dtypes.ProcessModuleConfig) error {
	sourceDir := installer.props.PathResolver.AgentSharedBinaryDirForDigest(installer.props.ImageDigest)
	return processmoduleconfig.CreateAgentConfigDir(installer.fs, targetDir, sourceDir, processModuleConfig)
}

func (installer *ImageInstaller) installAgentFromImage() error {
	defer func() { _ = installer.fs.RemoveAll(CacheDir) }()
	_ = installer.fs.MkdirAll(CacheDir, 0755)
	image := installer.props.ImageUri

	sourceCtx, sourceRef, err := getSourceInfo(CacheDir, *installer.props)
	if err != nil {
		log.Info("failed to get source information", "image", image)
		return err
	}

	imageDigest, err := getImageDigest(sourceCtx, sourceRef)
	if err != nil {
		log.Info("failed to get image digest", "image", image)
		return err
	}

	imageDigestEncoded := imageDigest.Encoded()
	imageCacheDir := filepath.Join(CacheDir, imageDigestEncoded)
	if installer.isAlreadyDownloaded(imageDigestEncoded, imageCacheDir) {
		log.Info("image is already installed", "image", image, "digest", imageDigestEncoded)
		installer.props.ImageDigest = imageDigestEncoded
		return nil
	}
	sharedDir := installer.props.PathResolver.AgentSharedBinaryDirForDigest(imageDigestEncoded)
	_ = installer.fs.MkdirAll(sharedDir, 0755)

	destinationCtx, destinationRef, err := getDestinationInfo(imageCacheDir)
	if err != nil {
		log.Info("failed to get destination information", "image", image, "imageCacheDir", imageCacheDir)
		return err
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
		return err
	}
	installer.props.ImageDigest = imageDigestEncoded
	return nil
}

func (installer *ImageInstaller) isAlreadyDownloaded(imageDigestEncoded string, imageCacheDir string) bool {
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

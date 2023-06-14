package image

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Dynatrace/dynatrace-operator/src/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/src/dockerconfig"
	dtypes "github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/installer/common"
	"github.com/Dynatrace/dynatrace-operator/src/installer/symlink"
	"github.com/Dynatrace/dynatrace-operator/src/installer/zip"
	"github.com/Dynatrace/dynatrace-operator/src/processmoduleconfig"
	"github.com/containers/image/v5/docker/reference"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

type Properties struct {
	ImageUri     string
	DockerConfig dockerconfig.DockerConfig
	PathResolver metadata.PathResolver
	Metadata     metadata.Access
}

func NewImageInstaller(fs afero.Fs, props *Properties) (*Installer, error) {
	ref, err := reference.Parse(props.ImageUri)
	if err != nil {
		return nil, errors.WithMessage(err, fmt.Sprintf("failed to parse image reference to create image installer, received imageUri: %s", props.ImageUri))
	}
	canonRef, ok := ref.(reference.Canonical)
	if !ok {
		return nil, errors.Errorf("unexpected type of image reference provided to image installer, expected reference with digest but received %s", props.ImageUri)
	}
	return &Installer{
		fs:          fs,
		extractor:   zip.NewOneAgentExtractor(fs, props.PathResolver),
		props:       props,
		imageDigest: canonRef.Digest().Encoded(),
	}, nil
}

type Installer struct {
	fs        afero.Fs
	extractor zip.Extractor
	props     *Properties

	imageDigest string
}

func (installer Installer) ImageDigest() string {
	return installer.imageDigest
}

func (installer *Installer) InstallAgent(targetDir string) (bool, error) {
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

func (installer Installer) UpdateProcessModuleConfig(targetDir string, processModuleConfig *dtypes.ProcessModuleConfig) error {
	sourceDir := installer.props.PathResolver.AgentSharedBinaryDirForImage(installer.ImageDigest())
	return processmoduleconfig.CreateAgentConfigDir(installer.fs, targetDir, sourceDir, processModuleConfig)
}

func (installer Installer) Cleanup() error {
	return installer.props.DockerConfig.Cleanup(afero.Afero{Fs: installer.fs})
}

func (installer *Installer) installAgentFromImage() error {
	defer installer.fs.RemoveAll(CacheDir)
	err := installer.fs.MkdirAll(CacheDir, common.MkDirFileMode)
	if err != nil {
		log.Info("failed to create cache dir", "err", err)
		return errors.WithStack(err)
	}
	image := installer.props.ImageUri

	if installer.isAlreadyDownloaded(installer.ImageDigest()) {
		log.Info("image is already installed", "image", image, "digest", installer.ImageDigest())
		return nil
	}

	sourceCtx, sourceRef, err := getSourceInfo(CacheDir, *installer.props)
	if err != nil {
		log.Info("failed to get source information", "image", image)
		return errors.WithStack(err)
	}
	imageCacheDir := getCacheDirPath(installer.ImageDigest())
	destinationCtx, destinationRef, err := getDestinationInfo(imageCacheDir)
	if err != nil {
		log.Info("failed to get destination information", "image", image, "imageCacheDir", imageCacheDir)
		return errors.WithStack(err)
	}

	log.Info("docker", "dockerpath", installer.props.DockerConfig.RegistryAuthPath)

	err = installer.extractAgentBinariesFromImage(
		imagePullInfo{
			imageCacheDir:  imageCacheDir,
			targetDir:      installer.props.PathResolver.AgentSharedBinaryDirForImage(installer.ImageDigest()),
			sourceCtx:      sourceCtx,
			destinationCtx: destinationCtx,
			sourceRef:      sourceRef,
			destinationRef: destinationRef,
		},
		&installer.props.DockerConfig,
		installer.props.ImageUri,
	)
	if err != nil {
		log.Info("failed to extract agent binaries from image via proxy", "image", image, "imageCacheDir", imageCacheDir, "err", err)
	}

	return nil
}

func (installer Installer) isAlreadyDownloaded(imageDigestEncoded string) bool {
	sharedDir := installer.props.PathResolver.AgentSharedBinaryDirForImage(imageDigestEncoded)
	_, err := installer.fs.Stat(sharedDir)
	return !os.IsNotExist(err)
}

func getCacheDirPath(digest string) string {
	return filepath.Join(CacheDir, digest)
}

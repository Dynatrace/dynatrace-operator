package image

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Dynatrace/dynatrace-operator/src/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/src/dockerconfig"
	"github.com/Dynatrace/dynatrace-operator/src/installer"
	"github.com/Dynatrace/dynatrace-operator/src/installer/common"
	"github.com/Dynatrace/dynatrace-operator/src/installer/symlink"
	"github.com/Dynatrace/dynatrace-operator/src/installer/zip"
	"github.com/containers/image/v5/docker/reference"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

type Properties struct {
	ImageUri     string
	DockerConfig dockerconfig.DockerConfig
	PathResolver metadata.PathResolver
	Metadata     metadata.Access
	ImageDigest  string
}

func GetDigest(uri string) (string, error) {
	ref, err := reference.Parse(uri)
	if err != nil {
		return "", errors.WithMessage(err, fmt.Sprintf("failed to parse image reference to create image installer, received imageUri: %s", uri))
	}
	canonRef, ok := ref.(reference.Canonical)
	if !ok {
		return "", errors.Errorf("unexpected type of image reference provided to image installer, expected reference with digest but received %s", uri)
	}
	return canonRef.Digest().Encoded(), nil
}

func NewImageInstaller(fs afero.Fs, props *Properties) installer.Installer {
	return &Installer{
		fs:        fs,
		extractor: zip.NewOneAgentExtractor(fs, props.PathResolver),
		props:     props,
	}
}

type Installer struct {
	fs        afero.Fs
	extractor zip.Extractor
	props     *Properties
}

func (installer *Installer) InstallAgent(targetDir string) (bool, error) {
	log.Info("installing agent from image")

	if installer.isAlreadyPresent(targetDir) {
		log.Info("agent already installed", "target dir", targetDir)
		return false, nil
	}

	err := installer.fs.MkdirAll(installer.props.PathResolver.AgentSharedBinaryDirBase(), common.MkDirFileMode)
	if err != nil {
		log.Info("failed to create the base shared agent directory", "err", err)
		return false, errors.WithStack(err)
	}

	log.Info("installing agent", "target dir", targetDir)
	if err := installer.installAgentFromImage(targetDir); err != nil {
		_ = installer.fs.RemoveAll(targetDir)
		log.Info("failed to install agent from image", "err", err)
		return false, errors.WithStack(err)
	}

	if err := symlink.CreateSymlinkForCurrentVersionIfNotExists(installer.fs, targetDir); err != nil {
		_ = installer.fs.RemoveAll(targetDir)
		log.Info("failed to create symlink for agent installation", "err", err)
		return false, errors.WithStack(err)
	}
	return true, nil
}

func (installer Installer) Cleanup() error {
	return installer.props.DockerConfig.Cleanup(afero.Afero{Fs: installer.fs})
}

func (installer *Installer) installAgentFromImage(targetDir string) error {
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
	imageCacheDir := getCacheDirPath(installer.props.ImageDigest)
	destinationCtx, destinationRef, err := getDestinationInfo(imageCacheDir)
	if err != nil {
		log.Info("failed to get destination information", "image", image, "imageCacheDir", imageCacheDir)
		return errors.WithStack(err)
	}

	err = installer.extractAgentBinariesFromImage(
		imagePullInfo{
			imageCacheDir:  imageCacheDir,
			targetDir:      targetDir,
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
	return nil
}

func (installer Installer) isAlreadyPresent(targetDir string) bool {
	_, err := installer.fs.Stat(targetDir)
	return !os.IsNotExist(err)
}

func getCacheDirPath(digest string) string {
	return filepath.Join(CacheDir, digest)
}

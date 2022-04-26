package image

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/Dynatrace/dynatrace-operator/src/dockerconfig"
	dtypes "github.com/Dynatrace/dynatrace-operator/src/dtclient/types"
	"github.com/Dynatrace/dynatrace-operator/src/installer/symlink"
	"github.com/Dynatrace/dynatrace-operator/src/processmoduleconfig"
	"github.com/containers/image/v5/docker"
	"github.com/containers/image/v5/types"
	"github.com/opencontainers/go-digest"
	"github.com/spf13/afero"
)

type Properties struct {
	Image        string
	DockerConfig dockerconfig.DockerConfig
}

func NewImageInstaller(fs afero.Fs, props *Properties) *imageInstaller {
	return &imageInstaller{
		fs:    fs,
		props: props,
	}
}

type imageInstaller struct {
	fs    afero.Fs
	props *Properties
}

func (installer *imageInstaller) InstallAgent(targetDir string) error {
	log.Info("installing agent", "target dir", targetDir)
	if err := installer.installAgentFromImage(targetDir); err != nil {
		_ = installer.fs.RemoveAll(targetDir)
		return fmt.Errorf("failed to install agent: %w", err)
	}
	return symlink.CreateSymlinkIfNotExists(installer.fs, targetDir)
}

func (installer *imageInstaller) UpdateProcessModuleConfig(targetDir string, processModuleConfig *dtypes.ProcessModuleConfig) error {
	return processmoduleconfig.UpdateProcessModuleConfig(installer.fs, targetDir, processModuleConfig)
}

func (installer *imageInstaller) installAgentFromImage(targetDir string) error {
	_ = installer.fs.MkdirAll(CacheDir, 0755)
	image := installer.props.Image

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
	imageCacheDir := filepath.Join(CacheDir, imageDigest.Encoded())

	destinationCtx, destinationRef, err := getDestinationInfo(imageCacheDir)
	if err != nil {
		log.Info("failed to get destination information", "image", image, "imageCacheDir", imageCacheDir)
		return err
	}
	return installer.getAgentFromImage(
		imagePullInfo{
			imageCacheDir:  imageCacheDir,
			targetDir:      targetDir,
			sourceCtx:      sourceCtx,
			destinationCtx: destinationCtx,
			sourceRef:      sourceRef,
			destinationRef: destinationRef,
		},
	)

}

func getImageDigest(systemContext *types.SystemContext, imageReference *types.ImageReference) (digest.Digest, error) {
	return docker.GetDigest(context.TODO(), systemContext, *imageReference)
}

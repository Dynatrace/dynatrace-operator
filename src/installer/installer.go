package installer

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/Dynatrace/dynatrace-operator/src/dockerconfig"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/processmoduleconfig"
	"github.com/spf13/afero"
)

const (
	agentConfPath = "agent/conf/"
	VersionLatest = "latest"
)

var (
	ruxitAgentProcPath       = filepath.Join("agent", "conf", "ruxitagentproc.conf")
	sourceRuxitAgentProcPath = filepath.Join("agent", "conf", "_ruxitagentproc.conf")
)

type InstallerProperties struct {
	Os           string
	Arch         string
	Type         string
	Flavor       string
	Version      string
	Technologies []string
	Url          string         // if this is set all others will be ignored
	ImageInfo    *ImagePullInfo // if this is set all others will be ignored, overrules Url
}

type ImagePullInfo struct {
	Image        string
	DockerConfig dockerconfig.DockerConfig
}

func (props *InstallerProperties) fillEmptyWithDefaults() {
	if props.Technologies == nil || len(props.Technologies) == 0 {
		props.Technologies = []string{"all"}
	}
}

type Installer interface {
	InstallAgent(targetDir string) error
	UpdateProcessModuleConfig(targetDir string, processModuleConfig *dtclient.ProcessModuleConfig) error
	SetVersion(version string)
	SetImagePullInfo(imagePullInfo *ImagePullInfo)
}

var _ Installer = &OneAgentInstaller{}

type OneAgentInstaller struct {
	fs    afero.Fs
	dtc   dtclient.Client
	props InstallerProperties
}

func NewOneAgentInstaller(
	fs afero.Fs,
	dtc dtclient.Client,
	props InstallerProperties,
) *OneAgentInstaller {
	return &OneAgentInstaller{
		fs:    fs,
		dtc:   dtc,
		props: props,
	}
}

func (installer *OneAgentInstaller) InstallAgent(targetDir string) error {
	log.Info("installing agent", "target dir", targetDir)
	installer.props.fillEmptyWithDefaults()
	if installer.props.ImageInfo != nil {
		return installer.installAgentFromImage(targetDir)
	}
	if err := installer.installAgentFromTenant(targetDir); err != nil {
		_ = installer.fs.RemoveAll(targetDir)

		return fmt.Errorf("failed to install agent: %w", err)
	}

	return nil
}

func (installer *OneAgentInstaller) SetVersion(version string) {
	installer.props.Version = version
}

func (installer *OneAgentInstaller) SetImagePullInfo(imagePullInfo *ImagePullInfo) {
	installer.props.ImageInfo = imagePullInfo
}

func (installer *OneAgentInstaller) createSymlinkIfNotExists(targetDir string) error {
	fs := installer.fs

	// MemMapFs (used for testing) doesn't comply with the Linker interface
	linker, ok := fs.(afero.Linker)
	if !ok {
		log.Info("symlinking not possible", "version", installer.props.Version, "fs", installer.fs)
		return nil
	}

	relativeSymlinkPath := installer.props.Version
	symlinkTargetPath := filepath.Join(targetDir, "agent", "bin", "current")
	if fileInfo, _ := fs.Stat(symlinkTargetPath); fileInfo != nil {
		log.Info("symlink already exists", "location", symlinkTargetPath)
		return nil
	}

	log.Info("creating symlink", "points-to(relative)", relativeSymlinkPath, "location", symlinkTargetPath)
	if err := linker.SymlinkIfPossible(relativeSymlinkPath, symlinkTargetPath); err != nil {
		log.Info("symlinking failed", "version", installer.props.Version)
		return err
	}
	return nil
}

func (installer *OneAgentInstaller) UpdateProcessModuleConfig(targetDir string, processModuleConfig *dtclient.ProcessModuleConfig) error {
	if processModuleConfig != nil {
		log.Info("updating ruxitagentproc.conf", "agentVersion", installer.props.Version)
		usedProcessModuleConfigPath := filepath.Join(targetDir, ruxitAgentProcPath)
		sourceProcessModuleConfigPath := filepath.Join(targetDir, sourceRuxitAgentProcPath)
		if err := installer.checkProcessModuleConfigCopy(sourceProcessModuleConfigPath, usedProcessModuleConfigPath); err != nil {
			return err
		}
		return processmoduleconfig.Update(installer.fs, sourceProcessModuleConfigPath, usedProcessModuleConfigPath, processModuleConfig.ToMap())
	}
	log.Info("no changes to ruxitagentproc.conf, skipping update")
	return nil
}

// checkProcessModuleConfigCopy checks if we already made a copy of the original ruxitagentproc.conf file.
// After the initial install of a version we copy the ruxitagentproc.conf to _ruxitagentproc.conf and we use the _ruxitagentproc.conf + the api response to re-create the ruxitagentproc.conf
// so its easier to update
func (installer *OneAgentInstaller) checkProcessModuleConfigCopy(sourcePath, destPath string) error {
	if _, err := installer.fs.Open(sourcePath); os.IsNotExist(err) {
		log.Info("saving original ruxitagentproc.conf to _ruxitagentproc.conf")
		fileInfo, err := installer.fs.Stat(destPath)
		if err != nil {
			return err
		}

		sourceProcessModuleConfigFile, err := installer.fs.OpenFile(sourcePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fileInfo.Mode())
		if err != nil {
			return err
		}

		usedProcessModuleConfigFile, err := installer.fs.Open(destPath)
		if err != nil {
			return err
		}
		_, err = io.Copy(sourceProcessModuleConfigFile, usedProcessModuleConfigFile)
		if err != nil {
			sourceProcessModuleConfigFile.Close()
			usedProcessModuleConfigFile.Close()
			return err
		}
		if err = sourceProcessModuleConfigFile.Close(); err != nil {
			return err
		}
		return usedProcessModuleConfigFile.Close()
	}
	return nil
}

package url

import (
	"os"

	"github.com/Dynatrace/dynatrace-operator/src/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/installer/symlink"
	"github.com/Dynatrace/dynatrace-operator/src/installer/zip"
	"github.com/Dynatrace/dynatrace-operator/src/processmoduleconfig"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

type Properties struct {
	Os              string
	Arch            string
	Type            string
	Flavor          string
	TargetVersion   string
	PreviousVersion string
	Technologies    []string
	Url             string // if this is set all settings before it will be ignored

	PathResolver metadata.PathResolver
}

func (props *Properties) fillEmptyWithDefaults() {
	if props.Technologies == nil || len(props.Technologies) == 0 {
		props.Technologies = []string{"all"}
	}
}

type UrlInstaller struct {
	fs        afero.Fs
	dtc       dtclient.Client
	extractor zip.Extractor
	props     *Properties
}

func NewUrlInstaller(fs afero.Fs, dtc dtclient.Client, props *Properties) *UrlInstaller {
	return &UrlInstaller{
		fs:        fs,
		dtc:       dtc,
		extractor: zip.NewOnAgentExtractor(fs, props.PathResolver),
		props:     props,
	}
}

func (installer UrlInstaller) InstallAgent(targetDir string) (bool, error) {
	if installer.isAlreadyDownloaded(targetDir) {
		log.Info("agent already installed", "target dir", targetDir)
		return false, nil
	}
	log.Info("installing agent", "target dir", targetDir)
	installer.props.fillEmptyWithDefaults()
	if err := installer.installAgentFromUrl(targetDir); err != nil {
		_ = installer.fs.RemoveAll(targetDir)
		log.Info("failed to install agent", "targetDir", targetDir)
		return false, err
	}

	if err := symlink.CreateSymlinkForCurrentVersionIfNotExists(installer.fs, targetDir); err != nil {
		_ = installer.fs.RemoveAll(targetDir)
		log.Info("failed to create symlink for agent installation", "targetDir", targetDir)
		return false, err
	}
	return true, nil
}

func (installer UrlInstaller) UpdateProcessModuleConfig(targetDir string, processModuleConfig *dtclient.ProcessModuleConfig) error {
	return processmoduleconfig.UpdateProcessModuleConfigInPlace(installer.fs, targetDir, processModuleConfig)
}

func (installer UrlInstaller) installAgentFromUrl(targetDir string) error {
	fs := installer.fs
	tmpFile, err := afero.TempFile(fs, "", "download")
	if err != nil {
		log.Info("failed to create temp file download", "err", err)
		return errors.WithStack(err)
	}
	defer func() {
		_ = tmpFile.Close()
		if err := fs.Remove(tmpFile.Name()); err != nil {
			log.Error(err, "failed to delete downloaded file", "path", tmpFile.Name())
		}
	}()
	if err := installer.downloadOneAgentFromUrl(tmpFile); err != nil {
		return err
	}
	return installer.unpackOneAgentZip(targetDir, tmpFile)
}

func (installer UrlInstaller) isAlreadyDownloaded(targetDir string) bool {
	if standaloneBinDir == targetDir {
		return false
	}
	_, err := installer.fs.Stat(targetDir)
	return !os.IsNotExist(err)
}

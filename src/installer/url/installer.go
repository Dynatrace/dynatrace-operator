package url

import (
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient/types"
	"github.com/Dynatrace/dynatrace-operator/src/installer/symlink"
	"github.com/Dynatrace/dynatrace-operator/src/processmoduleconfig"
	"github.com/spf13/afero"
)

type Properties struct {
	Os           string
	Arch         string
	Type         string
	Flavor       string
	Version      string
	Technologies []string
	Url          string // if this is set all settings before it will be ignored
}

func (props *Properties) fillEmptyWithDefaults() {
	if props.Technologies == nil || len(props.Technologies) == 0 {
		props.Technologies = []string{"all"}
	}
}

func NewUrlInstaller(fs afero.Fs, dtc dtclient.Client, props *Properties) *urlInstaller {
	return &urlInstaller{
		fs:    fs,
		dtc:   dtc,
		props: props,
	}
}

type urlInstaller struct {
	fs    afero.Fs
	dtc   dtclient.Client
	props *Properties
}

func (installer *urlInstaller) InstallAgent(targetDir string) error {
	log.Info("installing agent", "target dir", targetDir)
	installer.props.fillEmptyWithDefaults()
	if err := installer.installAgentFromUrl(targetDir); err != nil {
		_ = installer.fs.RemoveAll(targetDir)
		return fmt.Errorf("failed to install agent: %w", err)
	}
	return symlink.CreateSymlinkIfNotExists(installer.fs, targetDir)
}

func (installer *urlInstaller) UpdateProcessModuleConfig(targetDir string, processModuleConfig *types.ProcessModuleConfig) error {
	return processmoduleconfig.UpdateProcessModuleConfig(installer.fs, targetDir, processModuleConfig)
}

func (installer *urlInstaller) installAgentFromUrl(targetDir string) error {
	fs := installer.fs
	tmpFile, err := afero.TempFile(fs, "", "download")
	if err != nil {
		return fmt.Errorf("failed to create temporary file for download: %w", err)
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

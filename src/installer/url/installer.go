package url

import (
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/installer/symlink"
	"github.com/Dynatrace/dynatrace-operator/src/processmoduleconfig"
	"github.com/pkg/errors"
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

type UrlInstaller struct {
	fs    afero.Fs
	dtc   dtclient.Client
	props *Properties
}

func NewUrlInstaller(fs afero.Fs, dtc dtclient.Client, props *Properties) *UrlInstaller {
	return &UrlInstaller{
		fs:    fs,
		dtc:   dtc,
		props: props,
	}
}

func (installer UrlInstaller) InstallAgent(targetDir string) error {
	log.Info("installing agent", "target dir", targetDir)
	installer.props.fillEmptyWithDefaults()
	err := installer.fs.MkdirAll(targetDir, 0755)
	if err != nil {
		log.Info("failed to create install target dir", "err", err, "targetDir", targetDir)
		return errors.WithStack(err)
	}
	if err := installer.installAgentFromUrl(targetDir); err != nil {
		_ = installer.fs.RemoveAll(targetDir)
		log.Info("failed to install agent", "err", err, "targetDir", targetDir)
		return errors.WithStack(err)
	}
	return symlink.CreateSymlinkForCurrentVersionIfNotExists(installer.fs, targetDir)
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
		return errors.WithStack(err)
	}
	return installer.unpackOneAgentZip(targetDir, tmpFile)
}

package installer

import (
	"fmt"
	"strings"

	"github.com/spf13/afero"
)

func (installer *OneAgentInstaller) installAgentFromUrl(targetDir string) error {
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

func (installer *OneAgentInstaller) unpackOneAgentZip(targetDir string, tmpFile afero.File) error {
	var fileSize int64
	if stat, err := tmpFile.Stat(); err == nil {
		fileSize = stat.Size()
	}

	log.Info("saved OneAgent package", "dest", tmpFile.Name(), "size", fileSize)
	log.Info("unzipping OneAgent package")
	if err := extractZip(installer.fs, tmpFile, targetDir); err != nil {
		return fmt.Errorf("failed to unzip file: %w", err)
	}
	log.Info("unzipped OneAgent package")
	return nil
}

func (installer *OneAgentInstaller) downloadOneAgentFromUrl(tmpFile afero.File) error {
	if installer.props.Url != "" {
		if err := installer.downloadOneAgentViaInstallerUrl(tmpFile); err != nil {
			return err
		}
	} else if installer.props.Version == VersionLatest {
		if err := installer.downloadLatestOneAgent(tmpFile); err != nil {
			return err
		}
	} else {
		if err := installer.downloadOneAgentWithVersion(tmpFile); err != nil {
			return err
		}
	}
	return nil
}

func (installer *OneAgentInstaller) downloadLatestOneAgent(tmpFile afero.File) error {
	log.Info("downloading latest OneAgent package", "props", installer.props)
	return installer.dtc.GetLatestAgent(
		installer.props.Os,
		installer.props.Type,
		installer.props.Flavor,
		installer.props.Arch,
		installer.props.Technologies,
		tmpFile,
	)
}

func (installer *OneAgentInstaller) downloadOneAgentWithVersion(tmpFile afero.File) error {
	log.Info("downloading specific OneAgent package", "version", installer.props.Version)
	err := installer.dtc.GetAgent(
		installer.props.Os,
		installer.props.Type,
		installer.props.Flavor,
		installer.props.Arch,
		installer.props.Version,
		installer.props.Technologies,
		tmpFile,
	)

	if err != nil {
		availableVersions, getVersionsError := installer.dtc.GetAgentVersions(
			installer.props.Os,
			installer.props.Type,
			installer.props.Flavor,
			installer.props.Arch,
		)
		if getVersionsError != nil {
			return fmt.Errorf("failed to fetch OneAgent version: %w", err)
		}
		return fmt.Errorf("failed to fetch OneAgent version: %w, available versions are: %s", err, "[ "+strings.Join(availableVersions, " , ")+" ]")
	}
	return nil
}

func (installer *OneAgentInstaller) downloadOneAgentViaInstallerUrl(tmpFile afero.File) error {
	log.Info("downloading OneAgent package using provided url, all other properties are ignored", "url", installer.props.Url)
	return installer.dtc.GetAgentViaInstallerUrl(installer.props.Url, tmpFile)
}

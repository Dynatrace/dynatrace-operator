package url

import (
	"fmt"
	"strings"

	"github.com/spf13/afero"
)

func (installer *UrlInstaller) downloadOneAgentFromUrl(tmpFile afero.File) error {
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

func (installer *UrlInstaller) downloadLatestOneAgent(tmpFile afero.File) error {
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

func (installer *UrlInstaller) downloadOneAgentWithVersion(tmpFile afero.File) error {
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

func (installer *UrlInstaller) downloadOneAgentViaInstallerUrl(tmpFile afero.File) error {
	log.Info("downloading OneAgent package using provided url, all other properties are ignored", "url", installer.props.Url)
	return installer.dtc.GetAgentViaInstallerUrl(installer.props.Url, tmpFile)
}

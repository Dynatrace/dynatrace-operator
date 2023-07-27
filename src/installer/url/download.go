package url

import (
	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

func (installer Installer) downloadOneAgentFromUrl(tmpFile afero.File) error {
	switch {
	case installer.props.Url != "":
		if err := installer.downloadOneAgentViaInstallerUrl(tmpFile); err != nil {
			return errors.WithStack(err)
		}
	case installer.props.TargetVersion == VersionLatest:
		if err := installer.downloadLatestOneAgent(tmpFile); err != nil {
			return errors.WithStack(err)
		}
	default:
		if err := installer.downloadOneAgentWithVersion(tmpFile); err != nil {
			return err
		}
	}
	return nil
}

func (installer Installer) downloadLatestOneAgent(tmpFile afero.File) error {
	log.Info("downloading latest OneAgent package", "props", installer.props)
	return installer.dtc.GetLatestAgent(
		installer.props.Os,
		installer.props.Type,
		installer.props.Flavor,
		installer.props.Arch,
		installer.props.Technologies,
		installer.props.SkipMetadata,
		tmpFile,
	)
}

func (installer Installer) downloadOneAgentWithVersion(tmpFile afero.File) error {
	log.Info("downloading specific OneAgent package", "version", installer.props.TargetVersion)
	err := installer.dtc.GetAgent(
		installer.props.Os,
		installer.props.Type,
		installer.props.Flavor,
		installer.props.Arch,
		installer.props.TargetVersion,
		installer.props.Technologies,
		installer.props.SkipMetadata,
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
			log.Info("failed to get available versions", "err", getVersionsError)
			return errors.WithStack(getVersionsError)
		}
		log.Info("failed to download specific OneAgent package", "version", installer.props.TargetVersion, "available versions", availableVersions)
		return errors.WithStack(err)
	}
	return nil
}

func (installer Installer) downloadOneAgentViaInstallerUrl(tmpFile afero.File) error {
	log.Info("downloading OneAgent package using provided url, all other properties are ignored", "url", installer.props.Url)
	return installer.dtc.GetAgentViaInstallerUrl(installer.props.Url, tmpFile)
}

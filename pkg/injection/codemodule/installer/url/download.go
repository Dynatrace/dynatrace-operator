package url

import (
	"context"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"io"
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
	resp, err := installer.dtc.DeploymentAPI.DownloadLatestAgentInstaller(context.TODO(), installer.props.Os, installer.props.Type).Arch(installer.props.Arch).Flavor(installer.props.Flavor).SkipMetadata(installer.props.SkipMetadata).Include(installer.props.Technologies).Execute()
	if err != nil {
		return err
	}
	_, err = io.Copy(tmpFile, resp.Body)
	return err
}

func (installer Installer) downloadOneAgentWithVersion(tmpFile afero.File) error {
	log.Info("downloading specific OneAgent package", "version", installer.props.TargetVersion)
	resp, err := installer.dtc.DeploymentAPI.DownloadAgentInstallerWithVersion(context.TODO(), installer.props.Os, installer.props.Type, installer.props.TargetVersion).Arch(installer.props.Arch).Flavor(installer.props.Flavor).SkipMetadata(installer.props.SkipMetadata).Include(installer.props.Technologies).Execute()
	if err != nil {
		return err
	}
	_, err = io.Copy(tmpFile, resp.Body)
	return err

	// POC, so no
	//if err != nil {
	//	availableVersions, getVersionsError := installer.dtc.GetAgentVersions(
	//		installer.props.Os,
	//		installer.props.Type,
	//		installer.props.Flavor,
	//		installer.props.Arch,
	//	)
	//	if getVersionsError != nil {
	//		log.Info("failed to get available versions", "err", getVersionsError)
	//		return errors.WithStack(getVersionsError)
	//	}
	//	log.Info("failed to download specific OneAgent package", "version", installer.props.TargetVersion, "available versions", availableVersions)
	//	return errors.WithStack(err)
	//}
	//return nil
}

func (installer Installer) downloadOneAgentViaInstallerUrl(tmpFile afero.File) error {
	return nil
	// POC, so no
	//log.Info("downloading OneAgent package using provided url, all other properties are ignored", "url", installer.props.Url)
	//return installer.dtc.GetAgentViaInstallerUrl(installer.props.Url, tmpFile)
}

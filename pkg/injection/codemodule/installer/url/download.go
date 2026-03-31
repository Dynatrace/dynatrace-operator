package url

import (
	"context"
	"os"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/oneagent"
	"github.com/pkg/errors"
)

func (installer Installer) downloadOneAgentFromURL(ctx context.Context, tmpFile *os.File) error {
	switch {
	case installer.props.URL != "":
		if err := installer.downloadOneAgentViaInstallerURL(ctx, tmpFile); err != nil {
			return errors.WithStack(err)
		}
	case installer.props.TargetVersion == VersionLatest:
		if err := installer.downloadLatestOneAgent(ctx, tmpFile); err != nil {
			return errors.WithStack(err)
		}
	default:
		if err := installer.downloadOneAgentWithVersion(ctx, tmpFile); err != nil {
			return err
		}
	}

	return nil
}

func (installer Installer) downloadLatestOneAgent(ctx context.Context, tmpFile *os.File) error {
	log.Info("downloading latest OneAgent package", "props", installer.props)

	return installer.dtc.GetLatest(ctx, oneagent.GetParams{
		Os:            installer.props.Os,
		InstallerType: installer.props.Type,
		Flavor:        installer.props.Flavor,
		Technologies:  installer.props.Technologies,
		SkipMetadata:  installer.props.SkipMetadata,
	},
		tmpFile,
	)
}

func (installer Installer) downloadOneAgentWithVersion(ctx context.Context, tmpFile *os.File) error {
	log.Info("downloading specific OneAgent package", "version", installer.props.TargetVersion)

	err := installer.dtc.Get(ctx,
		oneagent.GetParams{
			Os:            installer.props.Os,
			InstallerType: installer.props.Type,
			Flavor:        installer.props.Flavor,
			Version:       installer.props.TargetVersion,
			Technologies:  installer.props.Technologies,
			SkipMetadata:  installer.props.SkipMetadata,
		},
		tmpFile,
	)
	if err != nil {
		availableVersions, getVersionsError := installer.dtc.GetVersions(ctx,
			oneagent.GetParams{
				Os:            installer.props.Os,
				InstallerType: installer.props.Type,
				Flavor:        installer.props.Flavor,
			},
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

func (installer Installer) downloadOneAgentViaInstallerURL(ctx context.Context, tmpFile *os.File) error {
	log.Info("downloading OneAgent package using provided url, all other properties are ignored", "url", installer.props.URL)

	return installer.dtc.GetViaInstallerURL(ctx, installer.props.URL, tmpFile)
}

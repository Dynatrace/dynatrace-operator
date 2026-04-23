package url

import (
	"context"
	"os"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/oneagent"
	"github.com/pkg/errors"
)

func (installer Installer) downloadOneAgentFromURL(ctx context.Context, tmpFile *os.File) error {
	if installer.props.TargetVersion == VersionLatest {
		return installer.downloadLatestOneAgent(ctx, tmpFile)
	}

	return installer.downloadOneAgentWithVersion(ctx, tmpFile)
}

func (installer Installer) downloadLatestOneAgent(ctx context.Context, tmpFile *os.File) error {
	log.Info("downloading latest OneAgent package", "props", installer.props)

	return installer.dtClient.GetLatest(ctx, oneagent.GetParams{
		OS:            installer.props.OS,
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

	err := installer.dtClient.Get(ctx,
		oneagent.GetParams{
			OS:            installer.props.OS,
			InstallerType: installer.props.Type,
			Flavor:        installer.props.Flavor,
			Version:       installer.props.TargetVersion,
			Technologies:  installer.props.Technologies,
			SkipMetadata:  installer.props.SkipMetadata,
		},
		tmpFile,
	)
	if err != nil {
		availableVersions, getVersionsError := installer.dtClient.GetVersions(ctx,
			oneagent.GetParams{
				OS:            installer.props.OS,
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

package binary

import (
	"context"
	"os"
	"path/filepath"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/common"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/symlink"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/zip"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/pkg/errors"
)

type Properties struct {
	OS            string
	Arch          string
	Type          string
	Flavor        string
	TargetVersion string

	PathResolver metadata.PathResolver
	Technologies []string
	SkipMetadata bool
}

func (props *Properties) fillEmptyWithDefaults() {
	if len(props.Technologies) == 0 {
		props.Technologies = []string{"all"}
	}
}

type Installer struct {
	dtClient  oneagent.Client
	extractor zip.Extractor
	props     *Properties
}

type NewFunc func(oneagent.Client, *Properties) installer.Installer

func NewInstaller(dtClient oneagent.Client, props *Properties) installer.Installer {
	return &Installer{
		dtClient:  dtClient,
		extractor: zip.NewOneAgentExtractor(props.PathResolver),
		props:     props,
	}
}

func (installer Installer) InstallAgent(ctx context.Context, targetDir string) (bool, error) {
	ctx, log := logd.NewFromContext(ctx, "oneagent-url")
	log.Info("installing agent from url")

	if installer.isAlreadyDownloaded(targetDir) {
		log.Info("agent already installed", "target dir", targetDir)

		return true, nil
	}

	err := os.MkdirAll(installer.props.PathResolver.AgentSharedBinaryDirBase(), common.MkDirFileMode)
	if err != nil {
		log.Info("failed to create the base shared agent directory", "err", err)

		return false, errors.WithStack(err)
	}

	log.Info("installing agent", "target dir", targetDir)
	installer.props.fillEmptyWithDefaults()

	if err := installer.installAgent(ctx, targetDir); err != nil {
		_ = os.RemoveAll(targetDir)
		log.Info("failed to install agent", "targetDir", targetDir)

		return false, err
	}

	if err := symlink.CreateForCurrentVersionIfNotExists(ctx, targetDir); err != nil {
		_ = os.RemoveAll(targetDir)
		log.Info("failed to create symlink for agent installation", "targetDir", targetDir)

		return false, err
	}

	return true, nil
}

func (installer Installer) installAgent(ctx context.Context, targetDir string) error {
	log := logd.FromContext(ctx)

	var path string
	if installer.isInitContainerMode() {
		path = targetDir
	} else {
		path = filepath.Dir(targetDir)
	}

	tmpFile, err := os.CreateTemp(path, "download")
	if err != nil {
		log.Info("failed to create temp file download", "err", err)

		return errors.WithStack(err)
	}

	defer func() {
		_ = tmpFile.Close()

		if err := os.Remove(tmpFile.Name()); err != nil {
			log.Error(err, "failed to delete downloaded file", "path", tmpFile.Name())
		}
	}()

	if err := installer.downloadOneAgent(ctx, tmpFile); err != nil {
		return err
	}

	return installer.unpackOneAgentZip(ctx, targetDir, tmpFile)
}

func (installer Installer) isInitContainerMode() bool {
	if installer.props != nil {
		return installer.props.PathResolver.RootDir == consts.AgentInitBinDirMount
	}

	return false
}

func (installer Installer) isAlreadyDownloaded(targetDir string) bool {
	if isStandaloneInstall(targetDir) {
		return false
	}

	_, err := os.Stat(targetDir)

	return !os.IsNotExist(err)
}

func (installer Installer) downloadOneAgent(ctx context.Context, tmpFile *os.File) error {
	log := logd.FromContext(ctx)

	if installer.props.TargetVersion == VersionLatest {
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

func isStandaloneInstall(targetDir string) bool {
	return consts.AgentInitBinDirMount == targetDir
}

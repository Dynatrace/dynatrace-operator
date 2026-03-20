package url

import (
	"context"
	"os"
	"path/filepath"

	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/common"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/symlink"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/zip"
	"github.com/pkg/errors"
)

type Properties struct {
	Os            string
	Arch          string
	Type          string
	Flavor        string
	TargetVersion string
	URL           string // if this is set all settings before it will be ignored

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
	dtc       dtclient.Client
	extractor zip.Extractor
	props     *Properties
}

type NewFunc func(dtclient.Client, *Properties) installer.Installer

var _ NewFunc = NewURLInstaller

func NewURLInstaller(dtc dtclient.Client, props *Properties) installer.Installer {
	return &Installer{
		dtc:       dtc,
		extractor: zip.NewOneAgentExtractor(props.PathResolver),
		props:     props,
	}
}

func (installer Installer) InstallAgent(ctx context.Context, targetDir string) (bool, error) {
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

	if err := symlink.CreateForCurrentVersionIfNotExists(targetDir); err != nil {
		_ = os.RemoveAll(targetDir)
		log.Info("failed to create symlink for agent installation", "targetDir", targetDir)

		return false, err
	}

	return true, nil
}

func (installer Installer) installAgent(ctx context.Context, targetDir string) error {
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

	if err := installer.downloadOneAgentFromURL(ctx, tmpFile); err != nil {
		return err
	}

	return installer.unpackOneAgentZip(targetDir, tmpFile)
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

func isStandaloneInstall(targetDir string) bool {
	return consts.AgentInitBinDirMount == targetDir
}

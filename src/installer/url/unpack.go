package url

import (
	"github.com/Dynatrace/dynatrace-operator/src/installer/zip"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

func (installer *UrlInstaller) unpackOneAgentZip(targetDir string, tmpFile afero.File) error {
	var fileSize int64
	if stat, err := tmpFile.Stat(); err == nil {
		fileSize = stat.Size()
	}

	log.Info("saved OneAgent package", "dest", tmpFile.Name(), "size", fileSize)
	log.Info("unzipping OneAgent package")
	if err := zip.ExtractZip(installer.fs, tmpFile, targetDir); err != nil {
		log.Info("failed to unzip OneAgent package", "err", err)
		return errors.WithStack(err)
	}
	log.Info("unzipped OneAgent package")
	return nil
}

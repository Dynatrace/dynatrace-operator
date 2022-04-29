package url

import (
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/src/installer/zip"
	"github.com/spf13/afero"
)

func (installer *urlInstaller) unpackOneAgentZip(targetDir string, tmpFile afero.File) error {
	var fileSize int64
	if stat, err := tmpFile.Stat(); err == nil {
		fileSize = stat.Size()
	}

	log.Info("saved OneAgent package", "dest", tmpFile.Name(), "size", fileSize)
	log.Info("unzipping OneAgent package")
	if err := zip.ExtractZip(installer.fs, tmpFile, targetDir); err != nil {
		return fmt.Errorf("failed to unzip file: %w", err)
	}
	log.Info("unzipped OneAgent package")
	return nil
}

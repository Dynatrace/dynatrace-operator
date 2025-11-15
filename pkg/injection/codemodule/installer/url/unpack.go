package url

import (
	"os"

	"github.com/pkg/errors"
)

func (installer Installer) unpackOneAgentZip(targetDir string, tmpFile *os.File) error {
	var fileSize int64
	if stat, err := tmpFile.Stat(); err == nil {
		fileSize = stat.Size()
	}

	log.Info("saved OneAgent package", "dest", tmpFile.Name(), "size", fileSize)
	log.Info("unzipping OneAgent package")

	if err := installer.extractor.ExtractZip(tmpFile, targetDir); err != nil {
		log.Info("failed to unzip OneAgent package", "err", err)

		return errors.WithStack(err)
	}

	log.Info("unzipped OneAgent package")

	return nil
}

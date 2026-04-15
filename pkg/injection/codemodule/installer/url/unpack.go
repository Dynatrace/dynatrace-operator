package url

import (
	"context"
	"os"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/pkg/errors"
)

func (installer Installer) unpackOneAgentZip(ctx context.Context, targetDir string, tmpFile *os.File) error {
	log := logd.FromContext(ctx)

	var fileSize int64
	if stat, err := tmpFile.Stat(); err == nil {
		fileSize = stat.Size()
	}

	log.Info("saved OneAgent package", "dest", tmpFile.Name(), "size", fileSize)
	log.Info("unzipping OneAgent package")

	if err := installer.extractor.ExtractZip(ctx, tmpFile, targetDir); err != nil {
		log.Info("failed to unzip OneAgent package", "err", err)

		return errors.WithStack(err)
	}

	log.Info("unzipped OneAgent package")

	return nil
}

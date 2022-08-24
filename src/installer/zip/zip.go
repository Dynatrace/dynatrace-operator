package zip

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/src/config"
	"github.com/Dynatrace/dynatrace-operator/src/installer/common"
	"github.com/klauspost/compress/zip"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

func (extractor OneAgentExtractor) ExtractZip(sourceFile afero.File, targetDir string) error {
	extractor.cleanTempZipDir()
	fs := extractor.fs
	if sourceFile == nil {
		return fmt.Errorf("file is nil")
	}

	fileInfo, err := sourceFile.Stat()
	if err != nil {
		log.Info("failed to get file info", "err", err)
		return errors.WithStack(err)
	}

	reader, err := zip.NewReader(sourceFile, fileInfo.Size())
	if err != nil {
		log.Info("failed to create zip reader", "err", err)
		return errors.WithStack(err)
	}

	extractDest := extractor.pathResolver.AgentTempUnzipDir()
	if extractor.pathResolver.RootDir == config.AgentBinDirMount {
		extractDest = targetDir
	}

	err = extractFilesFromZip(fs, extractDest, reader)
	if err != nil {
		log.Info("failed to extract files from zip", "err", err)
		return err
	}
	if extractDest != targetDir {
		err := extractor.moveToTargetDir(targetDir)
		if err != nil {
			log.Info("failed to move file to final destination", "err", err)
			return err
		}
	}
	return nil
}

func extractFilesFromZip(fs afero.Fs, targetDir string, reader *zip.Reader) error {
	if err := fs.MkdirAll(targetDir, common.MkDirFileMode); err != nil {
		return errors.WithStack(err)
	}
	for _, file := range reader.File {
		path := filepath.Join(targetDir, file.Name)

		// Check for ZipSlip: https://snyk.io/research/zip-slip-vulnerability
		if !strings.HasPrefix(path, filepath.Clean(targetDir)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", path)
		}

		mode := file.Mode()

		if file.FileInfo().IsDir() {
			err := fs.MkdirAll(path, mode)
			if err != nil {
				return errors.WithStack(err)
			}
			continue
		}

		if isAgentConfFile(file.Name) {
			mode = common.ReadWriteAllFileMode
		}

		if err := fs.MkdirAll(filepath.Dir(path), mode); err != nil {
			return errors.WithStack(err)
		}

		dstFile, err := fs.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
		if err != nil {
			return errors.WithStack(err)
		}

		srcFile, err := file.Open()
		if err != nil {
			return errors.WithStack(err)
		}

		_, err = io.Copy(dstFile, srcFile)
		if err != nil {
			return errors.WithStack(err)
		}
		dstFile.Close()
		srcFile.Close()
	}
	return nil
}

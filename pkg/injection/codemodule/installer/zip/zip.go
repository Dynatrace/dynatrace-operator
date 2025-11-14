package zip

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/common"
	"github.com/klauspost/compress/zip"
	"github.com/pkg/errors"
)

func (extractor OneAgentExtractor) ExtractZip(sourceFile *os.File, targetDir string) error {
	extractor.cleanTempZipDir()

	if sourceFile == nil {
		return errors.New("file is nil")
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

	extractDest := extractor.pathResolver.AgentTempUnzipRootDir()
	if extractor.pathResolver.RootDir == consts.AgentInitBinDirMount {
		extractDest = targetDir
	}

	err = extractFilesFromZip(extractDest, reader)
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

func extractFilesFromZip(targetDir string, reader *zip.Reader) error {
	if err := os.MkdirAll(targetDir, common.MkDirFileMode); err != nil {
		return errors.WithStack(err)
	}

	for _, file := range reader.File {
		path := filepath.Join(targetDir, file.Name)

		// Check for ZipSlip: https://snyk.io/research/zip-slip-vulnerability
		if !strings.HasPrefix(path, filepath.Clean(targetDir)+string(os.PathSeparator)) {
			return errors.Errorf("illegal file path: %s", path)
		}

		mode := file.Mode()

		if file.FileInfo().IsDir() {
			err := os.MkdirAll(path, mode)
			if err != nil {
				return errors.WithStack(err)
			}

			continue
		}

		if isAgentConfFile(file.Name) {
			mode = common.ReadWriteAllFileMode
		}

		if err := os.MkdirAll(filepath.Dir(path), mode); err != nil {
			return errors.WithStack(err)
		}

		dstFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
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

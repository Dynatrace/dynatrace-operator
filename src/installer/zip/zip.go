package zip

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/src/installer/common"
	"github.com/klauspost/compress/zip"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

func ExtractZip(fs afero.Fs, sourceFile afero.File, targetDir string) error {
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

	err = fs.MkdirAll(targetDir, common.MkDirFileMode)
	if err != nil {
		log.Info("failed to create target directory", "err", err)
		return errors.WithStack(err)
	}

	for _, file := range reader.File {
		if err := extractFileFromZip(fs, targetDir, file); err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

func extractFileFromZip(fs afero.Fs, targetDir string, file *zip.File) error {
	path := filepath.Join(targetDir, file.Name)

	// Check for ZipSlip: https://snyk.io/research/zip-slip-vulnerability
	if !strings.HasPrefix(path, filepath.Clean(targetDir)+string(os.PathSeparator)) {
		return fmt.Errorf("illegal file path: %s", path)
	}

	mode := file.Mode()

	if file.FileInfo().IsDir() {
		return fs.MkdirAll(path, mode)
	}

	if err := fs.MkdirAll(filepath.Dir(path), mode); err != nil {
		return errors.WithStack(err)
	}

	if isRuxitConfFile(file.Name) {
		mode = common.RuxitConfFileMode
	}

	dstFile, err := fs.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return errors.WithStack(err)
	}
	defer dstFile.Close()

	srcFile, err := file.Open()
	if err != nil {
		return errors.WithStack(err)
	}
	defer srcFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return errors.WithStack(err)
}

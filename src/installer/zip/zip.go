package zip

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/src/installer/common"
	"github.com/klauspost/compress/zip"
	"github.com/spf13/afero"
)

func ExtractZip(fs afero.Fs, sourceFile afero.File, targetDir string) error {
	if sourceFile == nil {
		return fmt.Errorf("file is nil")
	}

	fileInfo, err := sourceFile.Stat()
	if err != nil {
		return fmt.Errorf("unable to determine file info: %w", err)
	}

	reader, err := zip.NewReader(sourceFile, fileInfo.Size())
	if err != nil {
		return fmt.Errorf("failed to open ZIP file: %w", err)
	}

	_ = fs.MkdirAll(targetDir, 0755)

	for _, file := range reader.File {
		if err := extractFileFromZip(fs, targetDir, file); err != nil {
			return err
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
	if isAgentConfFile(file.Name) {
		mode = common.ReadWriteAllFileMode
	}

	if file.FileInfo().IsDir() {
		return fs.MkdirAll(path, mode)
	}

	if err := fs.MkdirAll(filepath.Dir(path), mode); err != nil {
		return err
	}

	dstFile, err := fs.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer func() { _ = dstFile.Close() }()

	srcFile, err := file.Open()
	if err != nil {
		return err
	}
	defer func() { _ = srcFile.Close() }()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

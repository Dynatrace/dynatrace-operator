package installer

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/klauspost/compress/gzip"
	"github.com/klauspost/compress/zip"
	"github.com/spf13/afero"
)

func extractZip(fs afero.Fs, sourceFile afero.File, targetDir string) error {
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

	// Mark all files inside ./agent/conf as group-writable
	if file.Name != agentConfPath && isAgentConfFile(file.Name) {
		mode |= 020
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

func extractGzip(fs afero.Fs, sourceFilePath, targetDir string) error {
	targetDir = filepath.Clean(targetDir)
	log.Info("extracting tar gzip", "source", sourceFilePath, "destinationDir", targetDir)

	reader, err := fs.Open(sourceFilePath)
	if err != nil {
		return err
	}
	defer func() { _ = reader.Close() }()

	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		return err
	}
	defer func() { _ = gzipReader.Close() }()

	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}

		target := filepath.Join(targetDir, header.Name)

		// Check for ZipSlip: https://snyk.io/research/zip-slip-vulnerability
		if !strings.HasPrefix(target, targetDir) {
			return fmt.Errorf("illegal file path: %s", target)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := fs.MkdirAll(target, 0755); err != nil {
				return err
			}

		case tar.TypeLink:
			// MemMapFs (used for testing) doesn't comply with the Linker interface, using os in testing causes problems
			_, ok := fs.(afero.Linker)
			if !ok {
				log.Info("symlinking not possible", "targetDir", targetDir, "fs", fs)
				continue
			}
			// Afero doesn't support Link, so we have to use os.Link
			if err := os.Link(filepath.Join(targetDir, header.Linkname), target); err != nil {
				return err
			}

		case tar.TypeSymlink:
			// MemMapFs (used for testing) doesn't comply with the Linker interface
			linker, ok := fs.(afero.Linker)
			if !ok {
				log.Info("symlinking not possible", "targetDir", targetDir, "fs", fs)
				continue
			}
			if err := linker.SymlinkIfPossible(header.Linkname, target); err != nil {
				return err
			}

		case tar.TypeReg:
			mode := header.Mode
			if isAgentConfFile(header.Name) {
				mode |= 020
			}
			destinationFile, err := fs.OpenFile(target, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.FileMode(mode))
			if err != nil {
				return err
			}

			if _, err := io.Copy(destinationFile, tarReader); err != nil {
				return err
			}
			_ = destinationFile.Close()

		default:
			log.Info("skipping special file", "name", header.Name)
		}
	}
}

// Checks if a file is under /agent/conf
// Different archive file implementations return different output, this function handles the differences
// zip.File.Name == "path/to/file"
// tar.Header.Name == "./path/to/file"
func isAgentConfFile(fileName string) bool {
	return strings.HasPrefix(fileName, "./"+agentConfPath) || strings.HasPrefix(fileName, agentConfPath)
}

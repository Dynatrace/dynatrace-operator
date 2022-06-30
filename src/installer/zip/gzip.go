package zip

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/src/installer/common"
	"github.com/klauspost/compress/gzip"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

func ExtractGzip(fs afero.Fs, sourceFilePath, targetDir string) error {
	targetDir = filepath.Clean(targetDir)
	log.Info("extracting tar gzip", "source", sourceFilePath, "destinationDir", targetDir)

	reader, err := fs.Open(sourceFilePath)
	if err != nil {
		return errors.WithStack(err)
	}
	defer reader.Close()

	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		return errors.WithStack(err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			return nil
		} else if err != nil {
			return errors.WithStack(err)
		}

		target := filepath.Join(targetDir, header.Name)

		// Check for ZipSlip: https://snyk.io/research/zip-slip-vulnerability
		if !strings.HasPrefix(target, targetDir) {
			return fmt.Errorf("illegal file path: %s", target)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := fs.MkdirAll(target, common.MkDirFileMode); err != nil {
				return errors.WithStack(err)
			}
		case tar.TypeLink:
			if err := extractLink(fs, targetDir, target, header); err != nil {
				return errors.WithStack(err)
			}
		case tar.TypeSymlink:
			if err := extractSymlink(fs, targetDir, target, header); err != nil {
				return errors.WithStack(err)
			}
		case tar.TypeReg:
			if err := extractFile(fs, target, header, tarReader); err != nil {
				return errors.WithStack(err)
			}
		default:
			log.Info("skipping special file", "name", header.Name)
		}
	}
}

func extractLink(fs afero.Fs, targetDir, target string, header *tar.Header) error {
	// MemMapFs (used for testing) doesn't comply with the Linker interface, using os in testing causes problems
	_, ok := fs.(afero.Linker)
	if !ok {
		log.Info("symlinking not possible", "targetDir", targetDir, "fs", fs)
		return nil
	}
	// Afero doesn't support Link, so we have to use os.Link
	if err := os.Link(filepath.Join(targetDir, header.Linkname), target); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func extractSymlink(fs afero.Fs, targetDir, target string, header *tar.Header) error {
	// MemMapFs (used for testing) doesn't comply with the Linker interface
	linker, ok := fs.(afero.Linker)
	if !ok {
		log.Info("symlinking not possible", "targetDir", targetDir, "fs", fs)
		return nil
	}
	if err := linker.SymlinkIfPossible(header.Linkname, target); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func extractFile(fs afero.Fs, target string, header *tar.Header, tarReader *tar.Reader) error {
	mode := header.Mode
	if isAgentConfFile(header.Name) {
		mode = common.ReadWriteAllFileMode
	}
	destinationFile, err := fs.OpenFile(target, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.FileMode(mode))
	defer (func() { _ = destinationFile.Close() })()
	if err != nil {
		return errors.WithStack(err)
	}

	if _, err := io.Copy(destinationFile, tarReader); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

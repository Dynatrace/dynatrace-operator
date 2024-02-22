package zip

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/common"
	"github.com/klauspost/compress/gzip"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

func (extractor OneAgentExtractor) ExtractGzip(sourceFilePath, targetDir string) error {
	extractor.cleanTempZipDir()
	fs := extractor.fs
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

	tmpUnzipDir := extractor.pathResolver.AgentTempUnzipRootDir()
	tarReader := tar.NewReader(gzipReader)

	err = extractFilesFromGzip(fs, tmpUnzipDir, tarReader)
	if err != nil {
		return err
	}

	return extractor.moveToTargetDir(targetDir)
}

func extractFilesFromGzip(fs afero.Fs, targetDir string, reader *tar.Reader) error {
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			return nil
		} else if err != nil {
			return errors.WithStack(err)
		}

		target := filepath.Join(targetDir, header.Name)

		// Check for ZipSlip: https://snyk.io/research/zip-slip-vulnerability
		if !strings.HasPrefix(target, targetDir) {
			return fmt.Errorf("illegal file path: %s", target)
		}

		err = extract(fs, targetDir, reader, header, target)
		if err != nil {
			return err
		}
	}
}

func extract(fs afero.Fs, targetDir string, reader *tar.Reader, header *tar.Header, target string) error {
	switch header.Typeflag {
	case tar.TypeDir:
		if err := fs.MkdirAll(target, header.FileInfo().Mode()); err != nil {
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
		if err := extractFile(fs, target, header, reader); err != nil {
			return errors.WithStack(err)
		}
	default:
		log.Info("skipping special file", "name", header.Name)
	}

	return nil
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
	mode := header.FileInfo().Mode()
	if isAgentConfFile(header.Name) {
		mode = common.ReadWriteAllFileMode
	}

	destinationFile, err := fs.OpenFile(target, os.O_CREATE|os.O_RDWR|os.O_TRUNC, mode)

	defer (func() { _ = destinationFile.Close() })()

	if err != nil {
		return errors.WithStack(err)
	}

	if _, err := io.Copy(destinationFile, tarReader); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

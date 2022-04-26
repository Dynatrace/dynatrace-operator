package zip

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/klauspost/compress/gzip"
	"github.com/spf13/afero"
)

func ExtractGzip(fs afero.Fs, sourceFilePath, targetDir string) error {
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

package zip

import (
	"archive/tar"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/common"
	"github.com/klauspost/compress/gzip"
	"github.com/pkg/errors"
)

func (extractor OneAgentExtractor) ExtractGzip(sourceFilePath, targetDir string) error {
	extractor.cleanTempZipDir()

	targetDir = filepath.Clean(targetDir)

	log.Info("extracting tar gzip", "source", sourceFilePath, "destinationDir", targetDir)

	reader, err := os.Open(sourceFilePath)
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

	err = extractFilesFromGzip(tmpUnzipDir, tarReader)
	if err != nil {
		return err
	}

	return extractor.moveToTargetDir(targetDir)
}

func extractFilesFromGzip(targetDir string, reader *tar.Reader) error {
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
			return errors.Errorf("illegal file path: %s", target)
		}

		err = extract(targetDir, reader, header, target)
		if err != nil {
			return err
		}
	}
}

func extract(targetDir string, reader *tar.Reader, header *tar.Header, target string) error {
	switch header.Typeflag {
	case tar.TypeDir:
		if err := os.MkdirAll(target, header.FileInfo().Mode()); err != nil {
			return errors.WithStack(err)
		}
	case tar.TypeLink:
		if err := extractLink(targetDir, target, header); err != nil {
			return errors.WithStack(err)
		}
	case tar.TypeSymlink:
		if err := extractSymlink(targetDir, target, header); err != nil {
			return errors.WithStack(err)
		}
	case tar.TypeReg:
		if err := extractFile(target, header, reader); err != nil {
			return errors.WithStack(err)
		}
	default:
		log.Info("skipping special file", "name", header.Name)
	}

	return nil
}

func extractLink(targetDir, target string, header *tar.Header) error {
	if err := os.Link(filepath.Join(targetDir, header.Linkname), target); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func extractSymlink(targetDir, target string, header *tar.Header) error {
	if isSafeToSymlink(header.Linkname, targetDir, target) && isSafeToSymlink(header.Name, targetDir, target) {
		if err := os.Symlink(header.Linkname, target); err != nil {
			return errors.WithStack(err)
		}
	} else {
		log.Info("found unsafe symlink that would point outside of the target dir", "linkName", header.Linkname)
	}

	return nil
}

// isSafeToSymlink checks that the provided relative symlink is NOT pointing outside of the `targetDir`
func isSafeToSymlink(symlink, targetDir, target string) bool {
	if filepath.IsAbs(symlink) {
		return false
	}

	finalPath := filepath.Join(target, symlink)
	relpath, err := filepath.Rel(targetDir, finalPath)

	return err == nil && !strings.HasPrefix(filepath.Clean(relpath), "..")
}

func extractFile(target string, header *tar.Header, tarReader *tar.Reader) error {
	mode := header.FileInfo().Mode()
	if isAgentConfFile(header.Name) {
		mode = common.ReadWriteAllFileMode
	}

	destinationFile, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR|os.O_TRUNC, mode)

	defer (func() { _ = destinationFile.Close() })()

	if err != nil {
		return errors.WithStack(err)
	}

	if _, err := io.Copy(destinationFile, tarReader); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

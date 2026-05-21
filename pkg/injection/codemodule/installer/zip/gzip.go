package zip

import (
	"archive/tar"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/common"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/klauspost/compress/gzip"
	"github.com/pkg/errors"
)

func (extractor OneAgentExtractor) ExtractGzip(ctx context.Context, sourceFilePath, targetDir string) error {
	ctx, log := logd.NewFromContext(ctx, "oneagent-zip")

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

	err = extractFilesFromGzip(ctx, tmpUnzipDir, tarReader)
	if err != nil {
		return err
	}

	return extractor.moveToTargetDir(ctx, targetDir)
}

func extractFilesFromGzip(ctx context.Context, targetDir string, reader *tar.Reader) error {
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			return nil
		} else if err != nil {
			return errors.WithStack(err)
		}

		target := evaluateTargetPath(targetDir, header.Name)

		if err := isTargetSafeToCreate(targetDir, header.Name, target); err != nil {
			return err
		}

		err = extract(ctx, targetDir, reader, header, target)
		if err != nil {
			return err
		}
	}
}

func isTargetSafeToCreate(targetDir, name, target string) error {
	if len(name) == 0 {
		return errors.Errorf("illegal empty file name")
	}

	// Check for ZipSlip: https://snyk.io/research/zip-slip-vulnerability
	if !strings.HasPrefix(target, targetDir) {
		return errors.Errorf("illegal file path: %s", target)
	}

	rel, err := filepath.Rel(targetDir, target)
	if err != nil {
		return err
	}

	if strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return errors.Errorf("%q is outside of %q", name, targetDir)
	}

	return nil
}

func evaluateTargetPath(targetDir, name string) string {
	return filepath.Join(targetDir, name)
}

func extract(ctx context.Context, targetDir string, reader *tar.Reader, header *tar.Header, target string) error {
	log := logd.FromContext(ctx)

	switch header.Typeflag {
	case tar.TypeDir:
		if err := os.MkdirAll(target, header.FileInfo().Mode()); err != nil {
			return errors.WithStack(err)
		}
	case tar.TypeLink:
		if err := extractLink(ctx, targetDir, target, header); err != nil {
			return errors.WithStack(err)
		}
	case tar.TypeSymlink:
		if err := extractSymlink(ctx, targetDir, target, header); err != nil {
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

func extractLink(ctx context.Context, targetDir, target string, header *tar.Header) error {
	log := logd.FromContext(ctx)

	if isHardlinkSafeToLink(header.Linkname, targetDir) {
		if err := os.Link(filepath.Join(targetDir, header.Linkname), target); err != nil {
			return errors.WithStack(err)
		}
	} else {
		log.Info("found unsafe link that would point outside of the target dir", "linkName", header.Linkname)
	}

	return nil
}

// isHardlinkSafeToLink checks that the provided relative hardlink is NOT pointing outside of the `targetDir`
func isHardlinkSafeToLink(link, targetDir string) bool {
	if filepath.IsAbs(link) {
		return false
	}

	if len(link) == 0 {
		return false
	}

	finalPath := filepath.Join(targetDir, link)
	relpath, err := filepath.Rel(targetDir, finalPath)

	return err == nil && !strings.HasPrefix(filepath.Clean(relpath), "..")
}

func extractSymlink(ctx context.Context, targetDir, target string, header *tar.Header) error {
	log := logd.FromContext(ctx)

	if isSymlinkSafeToLink(header.Linkname, targetDir, target) {
		if err := os.Symlink(header.Linkname, target); err != nil {
			return errors.WithStack(err)
		}
	} else {
		log.Info("found unsafe symlink that would point outside of the target dir", "linkName", header.Linkname)
	}

	return nil
}

// isSymlinkSafeToLink checks that the provided relative symbolic link is NOT pointing outside of the `targetDir`
func isSymlinkSafeToLink(symlink, targetDir, target string) bool {
	if filepath.IsAbs(symlink) {
		return false
	}

	if len(symlink) == 0 {
		return false
	}

	symlinkDir := filepath.Dir(target)
	finalPath := filepath.Join(symlinkDir, symlink)
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

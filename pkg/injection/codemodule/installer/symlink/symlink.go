package symlink

import (
	iofs "io/fs"
	"os"
	"path/filepath"
	"regexp"

	"github.com/pkg/errors"
)

func CreateForCurrentVersionIfNotExists(targetDir string) error {
	var err error

	targetBinDir := filepath.Join(targetDir, binDir)

	relativeSymlinkPath, err := findVersionFromFileSystem(targetBinDir)
	if err != nil {
		log.Info("failed to get the version from the filesystem", "targetDir", targetDir)

		return err
	}

	return Create(relativeSymlinkPath, filepath.Join(targetBinDir, "current"))
}

func Create(targetDir, symlinkDir string) error {
	// Check if the symlink already exists
	if fileInfo, _ := os.Stat(symlinkDir); fileInfo != nil {
		log.Info("symlink already exists", "location", symlinkDir)

		return nil
	}

	log.Info("creating symlink", "points-to(relative)", targetDir, "location", symlinkDir)

	if err := os.Symlink(targetDir, symlinkDir); err != nil {
		log.Info("symlinking failed", "source", targetDir)

		return errors.WithStack(err)
	}

	return nil
}

func Remove(symlinkPath string) error {
	if err := os.Remove(symlinkPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Info("failed to remove symlink", "path", symlinkPath)

		return err
	}

	return nil
}

func findVersionFromFileSystem(targetDir string) (string, error) {
	var version string

	walkFiles := func(path string, info iofs.FileInfo, err error) error {
		if info == nil {
			log.Info(
				"file does not exist, are you using a correct codeModules image?",
				"path", path)

			return iofs.ErrNotExist
		}

		if !info.IsDir() {
			return nil
		}

		folderName := filepath.Base(path)
		if regexp.MustCompile(versionRegexp).Match([]byte(folderName)) {
			log.Info("found version", "version", folderName)
			version = folderName

			return iofs.ErrExist
		}

		return nil
	}

	err := filepath.Walk(targetDir, walkFiles)
	if errors.Is(err, iofs.ErrNotExist) {
		return "", errors.WithStack(err)
	}

	return version, nil
}

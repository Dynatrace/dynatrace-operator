package symlink

import (
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

	entries, err := os.ReadDir(targetDir)
	if err != nil {
		return "", errors.WithStack(err)
	}

	for _, entry := range entries {
		if entry.IsDir() && regexp.MustCompile(versionRegexp).Match([]byte(entry.Name())) {
			log.Info("found version", "version", entry.Name())
			version = entry.Name()

			break
		}
	}

	return version, nil
}

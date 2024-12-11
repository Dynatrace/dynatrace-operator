package symlink

import (
	iofs "io/fs"
	"path/filepath"
	"regexp"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

func CreateForCurrentVersionIfNotExists(fs afero.Fs, targetDir string) error {
	var err error

	targetBinDir := filepath.Join(targetDir, binDir)

	relativeSymlinkPath, err := findVersionFromFileSystem(fs, targetBinDir)
	if err != nil {
		log.Info("failed to get the version from the filesystem", "targetDir", targetDir)

		return err
	}

	return Create(fs, relativeSymlinkPath, filepath.Join(targetBinDir, "current"))
}

func Create(fs afero.Fs, targetDir, symlinkDir string) error {
	// MemMapFs (used for testing) doesn't comply with the Linker interface
	linker, ok := fs.(afero.Linker)
	if !ok {
		log.Info("symlinking not possible", "targetDir", targetDir, "fs", fs)

		return nil
	}

	// Check if the symlink already exists
	if fileInfo, _ := fs.Stat(symlinkDir); fileInfo != nil {
		log.Info("symlink already exists", "location", symlinkDir)
		return nil
	}

	log.Info("creating symlink", "points-to(relative)", targetDir, "location", symlinkDir)

	if err := linker.SymlinkIfPossible(targetDir, symlinkDir); err != nil {
		log.Info("symlinking failed", "source", targetDir)
		return errors.WithStack(err)
	}

	return nil
}

func Remove(fs afero.Fs, symlinkPath string) error {
	if info, _ := fs.Stat(symlinkPath); info != nil {
		log.Info("symlink to directory exists, removing it to ensure proper reinstallation or reconfiguration", "directory", symlinkPath)

		if err := fs.Remove(symlinkPath); err != nil {
			return err
		}
	}

	return nil
}

func findVersionFromFileSystem(fs afero.Fs, targetDir string) (string, error) {
	var version string

	aferoFs := afero.Afero{
		Fs: fs,
	}
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

	err := aferoFs.Walk(targetDir, walkFiles)
	if errors.Is(err, iofs.ErrNotExist) {
		return "", errors.WithStack(err)
	}

	return version, nil
}

package symlink

import (
	iofs "io/fs"
	"path/filepath"
	"regexp"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

func CreateSymlinkForCurrentVersionIfNotExists(fs afero.Fs, targetDir string) error {
	var relativeSymlinkPath string
	var err error
	targetBindDir := filepath.Join(targetDir, binDir)

	// MemMapFs (used for testing) doesn't comply with the Linker interface
	linker, ok := fs.(afero.Linker)
	if !ok {
		log.Info("symlinking not possible", "targetDir", targetDir, "fs", fs)
		return nil
	}

	relativeSymlinkPath, err = findVersionFromFileSystem(fs, targetBindDir)
	if err != nil {
		log.Info("failed to get the version from the filesystem", "targetDir", targetDir)
		return err
	}

	symlinkTargetPath := filepath.Join(targetBindDir, "current")
	if fileInfo, _ := fs.Stat(symlinkTargetPath); fileInfo != nil {
		log.Info("symlink already exists", "location", symlinkTargetPath)
		return nil
	}

	log.Info("creating symlink", "points-to(relative)", relativeSymlinkPath, "location", symlinkTargetPath)
	if err := linker.SymlinkIfPossible(relativeSymlinkPath, symlinkTargetPath); err != nil {
		log.Info("symlinking failed", "version", relativeSymlinkPath)
		return errors.WithStack(err)
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
	if err := aferoFs.Walk(targetDir, walkFiles); err != nil && err != iofs.ErrExist {
		return "", errors.WithStack(err)
	}
	return version, nil
}

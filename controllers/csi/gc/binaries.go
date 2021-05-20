package csigc

import (
	"path/filepath"

	dtcsi "github.com/Dynatrace/dynatrace-operator/controllers/csi"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

func (gc *CSIGarbageCollector) runBinaryGarbageCollection(envID string, latestVersion string) error {
	fs := &afero.Afero{Fs: gc.fs}

	versionReferenceBasePath := filepath.Join(gc.opts.RootDir, dtcsi.DataPath, envID, dtcsi.GarbageCollectionPath)
	gc.logger.Info("gc path", "versionReferenceBasePath", versionReferenceBasePath, "latestVersion", latestVersion)
	versionReferences, err := fs.ReadDir(versionReferenceBasePath)
	if err != nil {
		exists, _ := fs.DirExists(versionReferenceBasePath)
		if !exists {
			gc.logger.Info("Garbage collector usage file could not be found")
			return nil
		}
		return errors.WithStack(err)
	}

	for _, version := range versionReferences {
		binaryPath := filepath.Join(gc.opts.RootDir, dtcsi.DataPath, envID, "bin", version.Name())
		gc.logger.Info("garbage collecting path", "binaryPath", binaryPath)

		if version.Name() == latestVersion {
			continue
		}
		podReferences, err := fs.ReadDir(filepath.Join(versionReferenceBasePath, version.Name()))
		if err != nil {
			return err
		}

		gc.logger.Info("garbage collecting path", "binaryPath", binaryPath)

		if len(podReferences) == 0 {
			gc.logger.Info("Garbage collector deleting unused version", "version", version.Name())
			err = fs.RemoveAll(binaryPath + "-default")
			if err != nil {
				gc.logger.Info("warning - failed to delete default binary path", "version", version.Name())
			}
			err = fs.RemoveAll(binaryPath + "-musl")
			if err != nil {
				gc.logger.Info("warning - failed to delete musl binary path", "version", version.Name())
			}
			err = fs.RemoveAll(filepath.Join(versionReferenceBasePath, version.Name()))
			if err != nil {
				gc.logger.Info("warning - failed to delete version reference base path", "version", version.Name())
			}
		}
	}

	return nil
}

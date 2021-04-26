package csigc

import (
	"os"
	"path/filepath"

	dtcsi "github.com/Dynatrace/dynatrace-operator/controllers/csi"
	"github.com/go-logr/logr"
)

func runBinaryGarbageCollection(logger logr.Logger, envID string, latestVersion string) error {
	gcPath := filepath.Join(dtcsi.DataPath, envID, "gc")
	gcDirs, err := os.ReadDir(gcPath)
	if err != nil {
		return err
	}

	for _, dir := range gcDirs {
		if dir.Name() == latestVersion {
			continue
		}
		subDirs, err := os.ReadDir(filepath.Join(gcPath, dir.Name()))
		if err != nil {
			return err
		}
		if len(subDirs) == 0 {
			logger.Info("Garbage collector deleting unused version", "version", dir.Name())
			err := os.RemoveAll(filepath.Join(dtcsi.DataPath, envID, "bin", dir.Name()+"-default"))
			if err != nil {
				return err
			}
			err = os.RemoveAll(filepath.Join(dtcsi.DataPath, envID, "bin", dir.Name()+"-musl"))
			if err != nil {
				return err
			}
			err = os.RemoveAll(filepath.Join(gcPath, dir.Name()))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

package csigc

import (
	"os"

	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/spf13/afero"
)

func (gc *CSIGarbageCollector) runBinaryGarbageCollection() {
	fs := &afero.Afero{Fs: gc.fs}

	gcRunsMetric.Inc()

	codeModules, err := gc.db.ListDeletedCodeModules()
	if err != nil {
		log.Error(err, "failed to read deleted codemodules")

		return
	}

	for _, codeModule := range codeModules {
		if !gc.time.Now().Time.After(codeModule.DeletedAt.Time.Add(safeRemovalThreshold)) {
			log.Info("skipping recently orphaned codemodule", "version", codeModule.Version, "location", codeModule.Location)

			continue
		}

		log.Info("cleaning up orphaned codemodule binary", "version", codeModule.Version, "location", codeModule.Location)
		removeUnusedVersion(fs, codeModule.Location)

		err := gc.db.PurgeCodeModule(&metadata.CodeModule{Version: codeModule.Version})
		if err != nil {
			log.Error(err, "failed to delete codemodule database entry")

			return
		}
	}
}

func removeUnusedVersion(fs *afero.Afero, binaryPath string) {
	size, _ := dirSize(fs, binaryPath)

	err := fs.RemoveAll(binaryPath)
	if err != nil {
		log.Error(err, "codemodule delete failed", "path", binaryPath)
	} else {
		foldersRemovedMetric.Inc()
		reclaimedMemoryMetric.Add(float64(size))
	}
}

func dirSize(fs *afero.Afero, path string) (int64, error) {
	var size int64

	err := fs.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			size += info.Size()
		}

		return err
	})

	return size, err
}

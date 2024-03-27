package csigc

import (
	"context"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

func (gc *CSIGarbageCollector) runBinaryGarbageCollection(ctx context.Context) {
	fs := &afero.Afero{Fs: gc.fs}

	gcRunsMetric.Inc()

	codeModules, err := gc.db.ReadCodeModules(ctx)
	if err != nil {
		log.Error(err, "failed to read codemodules")
		return
	}
	for _, codeModule := range codeModules {
		if gc.db.IsCodeModuleOrphaned(ctx, &codeModule) {
			removeUnusedVersion(fs, codeModule.Location)

			err := gc.db.DeleteCodeModule(ctx, &codeModule)
			if err != nil {
				log.Error(err, "failed to delete codemodule")
				return
			}
		}
	}
}

func (gc *CSIGarbageCollector) getStoredVersions(fs *afero.Afero, tenantUUID string) ([]string, error) {
	bins, err := fs.ReadDir(gc.path.AgentBinaryDir(tenantUUID))
	versions := make([]string, 0, len(bins))

	if os.IsNotExist(err) {
		log.Info("no versions stored")

		return versions, nil
	} else if err != nil {
		return nil, errors.WithStack(err)
	}

	for _, bin := range bins {
		versions = append(versions, bin.Name())
	}

	return versions, nil
}

func shouldDeleteVersion(version string, usedVersions map[string]bool) bool {
	return !usedVersions[version]
}

func removeUnusedVersion(fs *afero.Afero, binaryPath string) {
	size, _ := dirSize(fs, binaryPath)

	err := fs.RemoveAll(binaryPath)
	if err != nil {
		log.Info("delete failed", "path", binaryPath)
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

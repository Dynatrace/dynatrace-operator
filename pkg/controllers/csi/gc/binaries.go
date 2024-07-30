package csigc

import (
	"context"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

func (gc *CSIGarbageCollector) runBinaryGarbageCollection(ctx context.Context, tenantUUID string) {
	fs := &afero.Afero{Fs: gc.fs}

	gcRunsMetric.Inc()

	usedVersions, err := gc.db.GetUsedVersions(ctx, tenantUUID)
	if err != nil {
		log.Info("failed to get used versions", "error", err)

		return
	}

	log.Info("got all used versions (in deprecated location)", "tenantUUID", tenantUUID, "len(usedVersions)", len(usedVersions))

	storedVersions, err := gc.getStoredVersions(fs, tenantUUID)
	if err != nil {
		log.Info("failed to get stored versions", "error", err)

		return
	}

	log.Info("got all stored versions (in deprecated location)", "tenantUUID", tenantUUID, "len(storedVersions)", len(storedVersions))

	setAgentBins, err := gc.db.GetLatestVersions(ctx)
	if err != nil {
		log.Error(err, "failed to get the set image digests")
	}

	for _, version := range storedVersions {
		_, isPinnedVersion := setAgentBins[version]

		shouldDelete := shouldDeleteVersion(version, usedVersions) && !isPinnedVersion
		if !shouldDelete {
			log.Info("skipped, version should not be deleted", "version", version)

			continue
		}

		binaryPath := gc.path.AgentBinaryDirForVersion(tenantUUID, version)
		log.Info("deleting unused version (in deprecated location)", "version", version, "path", binaryPath)
		removeUnusedVersion(fs, binaryPath)
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

	log.Info("removed outdate CodeModule binary", "location", binaryPath)
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

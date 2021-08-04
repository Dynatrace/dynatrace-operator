package csigc

import (
	"os"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/afero"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	reclaimedMemoryMetric = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "dynatrace",
		Subsystem: "csi_driver",
		Name:      "gc_reclaimed",
		Help:      "Amount of memory reclaimed by the GC",
	})

	foldersRemovedMetric = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "dynatrace",
		Subsystem: "csi_driver",
		Name:      "gc_folder_rmv",
		Help:      "Number of folders deleted by the GC",
	})

	gcRunsMetric = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "dynatrace",
		Subsystem: "csi_driver",
		Name:      "gc_runs",
		Help:      "Number of GC runs",
	})
)

func init() {
	metrics.Registry.MustRegister(reclaimedMemoryMetric)
	metrics.Registry.MustRegister(foldersRemovedMetric)
	metrics.Registry.MustRegister(gcRunsMetric)
}

func (gc *CSIGarbageCollector) runBinaryGarbageCollection(tenantUUID string, latestVersion string) {
	fs := &afero.Afero{Fs: gc.fs}
	gcRunsMetric.Inc()

	usedVersions, err := gc.db.GetUsedVersions(tenantUUID)
	if err != nil {
		gc.logger.Info("failed to get used versions", "error", err)
		return
	}
	gc.logger.Info("got all used versions", "tenantUUID", tenantUUID, "len(usedVersions)", len(usedVersions))

	storedVersions, err := gc.getStoredVersions(fs, tenantUUID)
	if err != nil {
		gc.logger.Info("failed to get stored versions", "error", err)
		return
	}
	gc.logger.Info("got all stored versions", "tenantUUID", tenantUUID, "len(storedVersions)", len(storedVersions))

	for _, version := range storedVersions {
		shouldDelete := isNotLatestVersion(version, latestVersion, gc.logger) &&
			shouldDeleteVersion(version, usedVersions)

		if shouldDelete {
			binaryPath := gc.path.AgentBinaryDirForVersion(tenantUUID, version)
			gc.logger.Info("deleting unused version", "version", version, "path", binaryPath)

			removeUnusedVersion(fs, binaryPath, gc.logger)
		}
	}
}

func (gc *CSIGarbageCollector) getStoredVersions(fs *afero.Afero, tenantUUID string) ([]string, error) {
	versions := []string{}
	bins, err := fs.ReadDir(gc.path.AgentBinaryDir(tenantUUID))
	if err != nil {
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

func isNotLatestVersion(version string, latestVersion string, logger logr.Logger) bool {
	if version == latestVersion {
		logger.Info("skipped, is latest")
		return false
	}
	return true
}

func removeUnusedVersion(fs *afero.Afero, binaryPath string, logger logr.Logger) {
	size, _ := dirSize(fs, binaryPath)
	err := fs.RemoveAll(binaryPath)
	if err != nil {
		logger.Info("delete failed", "path", binaryPath)
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

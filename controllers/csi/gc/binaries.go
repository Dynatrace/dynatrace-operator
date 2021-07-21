package csigc

import (
	"os"
	"path/filepath"

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

	binaryDir := filepath.Join(gc.opts.RootDir, tenantUUID, "bin")
	usedVersions, err := gc.getUsedVersions(tenantUUID)
	if err != nil {
		gc.logger.Info("failed to get used versions", "error", err)
		return
	}
	storedVersions, err := gc.getStoredVersions(fs, binaryDir)
	if err != nil {
		gc.logger.Info("failed to get stored versions", "error", err)
		return
	}

	for _, version := range storedVersions {
		shouldDelete := isNotLatestVersion(version, latestVersion, gc.logger) &&
			shouldDeleteVersion(version, usedVersions)

		if shouldDelete {
			binaryPath := filepath.Join(binaryDir, version)
			gc.logger.Info("deleting unused version", "version", version, "path", binaryPath)

			removeUnusedVersion(fs, binaryPath, gc.logger)
		}
	}
}

func (gc *CSIGarbageCollector) getUsedVersions(tenantUUID string) (map[string]bool, error) {
	versions, err := gc.db.GetUsedVersions(tenantUUID)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return versions, nil
}

func (gc *CSIGarbageCollector) getStoredVersions(fs *afero.Afero, binaryDir string) ([]string, error) {
	versions := []string{}
	bins, err := fs.ReadDir(binaryDir)
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

package csigc

import (
	"os"
	"path/filepath"

	dtcsi "github.com/Dynatrace/dynatrace-operator/controllers/csi"
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

func (gc *CSIGarbageCollector) runBinaryGarbageCollection(tenantUUID string, latestVersion string) error {
	fs := &afero.Afero{Fs: gc.fs}
	logger := gc.logger.WithValues("tenant", tenantUUID, "latestVersion", latestVersion)

	versionReferencesBase := filepath.Join(gc.opts.RootDir, tenantUUID, dtcsi.GarbageCollectionPath)
	logger.Info("run garbage collection for binaries", "versionReferencesBase", versionReferencesBase)

	versionReferences, err := fs.ReadDir(versionReferencesBase)
	if err != nil {
		exists, _ := fs.DirExists(versionReferencesBase)
		if !exists {
			logger.Info("skipped, version reference base directory not exists", "path", versionReferencesBase)
			return nil
		}
		return errors.WithStack(err)
	}

	for _, fileInfo := range versionReferences {
		version := fileInfo.Name()
		references := filepath.Join(versionReferencesBase, version)

		shouldDelete := isNotLatestVersion(version, latestVersion, logger) &&
			shouldDeleteVersion(fs, references, logger.WithValues("version", version))

		if shouldDelete {
			binaryPath := filepath.Join(gc.opts.RootDir, tenantUUID, "bin", version)
			logger.Info("deleting unused version", "version", version, "path", binaryPath)

			removeUnusedVersion(fs, binaryPath, references, logger)
		}
	}
	gcRunsMetric.Inc()
	return nil
}

func shouldDeleteVersion(fs *afero.Afero, references string, logger logr.Logger) bool {
	podReferences, err := fs.ReadDir(references)
	if err != nil {
		logger.Error(err, "skipped, failed to get references")
		return false

	} else if len(podReferences) > 0 {
		logger.Info("skipped, in use", "references", len(podReferences))
		return false
	}

	return true
}

func isNotLatestVersion(version string, latestVersion string, logger logr.Logger) bool {
	if version == latestVersion {
		logger.Info("skipped, is latest")
		return false
	}

	return true
}

func removeUnusedVersion(fs *afero.Afero, binaryPath string, references string, logger logr.Logger) {
	size, _ := dirSize(fs, binaryPath)
	err := fs.RemoveAll(binaryPath)
	if err != nil {
		logger.Info("delete failed", "path", binaryPath)
	} else {
		foldersRemovedMetric.Inc()
		reclaimedMemoryMetric.Add(float64(size))
	}

	if err := fs.RemoveAll(references); err != nil {
		logger.Info("delete failed", "path", references)
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

package csigc

import (
	"path/filepath"
	"testing"

	dtcsi "github.com/Dynatrace/dynatrace-operator/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

const (
	tenantUUID = "asd12345"
	version_1  = "1"
	version_2  = "2"
	version_3  = "3"
	rootDir    = "/tmp"
)

var (
	versionReferenceBasePath = filepath.Join(rootDir, tenantUUID, dtcsi.GarbageCollectionPath)
)

func TestBinaryGarbageCollector_succeedsWhenVersionReferenceBaseDirectoryNotExists(t *testing.T) {
	resetMetrics()
	gc := newMockGarbageCollector()

	err := gc.runBinaryGarbageCollection(tenantUUID, version_1)

	assert.Equal(t, float64(1), testutil.ToFloat64(gcRunsMetric))
	assert.Equal(t, float64(0), testutil.ToFloat64(foldersRemovedMetric))
	assert.Equal(t, float64(0), testutil.ToFloat64(reclaimedMemoryMetric))

	assert.NoError(t, err)
}

func TestBinaryGarbageCollector_succeedsWhenNoVersionsAvailable(t *testing.T) {
	resetMetrics()
	gc := newMockGarbageCollector()
	_ = gc.fs.MkdirAll(versionReferenceBasePath, 0770)

	err := gc.runBinaryGarbageCollection(tenantUUID, version_1)

	assert.Equal(t, float64(1), testutil.ToFloat64(gcRunsMetric))
	assert.Equal(t, float64(0), testutil.ToFloat64(foldersRemovedMetric))
	assert.Equal(t, float64(0), testutil.ToFloat64(reclaimedMemoryMetric))

	assert.NoError(t, err)
}

func TestBinaryGarbageCollector_ignoresLatest(t *testing.T) {
	resetMetrics()
	gc := newMockGarbageCollector()
	gc.mockUnusedVersions(version_1)

	err := gc.runBinaryGarbageCollection(tenantUUID, version_1)

	assert.Equal(t, float64(1), testutil.ToFloat64(gcRunsMetric))
	assert.Equal(t, float64(0), testutil.ToFloat64(foldersRemovedMetric))
	assert.Equal(t, float64(0), testutil.ToFloat64(reclaimedMemoryMetric))

	assert.NoError(t, err)
	gc.assertVersionExists(t, version_1)
}

func TestBinaryGarbageCollector_removesUnused(t *testing.T) {
	resetMetrics()
	gc := newMockGarbageCollector()
	gc.mockUnusedVersions(version_1, version_2, version_3)

	err := gc.runBinaryGarbageCollection(tenantUUID, version_2)

	assert.Equal(t, float64(1), testutil.ToFloat64(gcRunsMetric))
	assert.Equal(t, float64(2), testutil.ToFloat64(foldersRemovedMetric))

	assert.NoError(t, err)
	gc.assertVersionNotExists(t, version_1, version_3)
}

func TestBinaryGarbageCollector_ignoresUsed(t *testing.T) {
	resetMetrics()
	gc := newMockGarbageCollector()
	gc.mockUsedVersions(version_1, version_2, version_3)

	err := gc.runBinaryGarbageCollection(tenantUUID, version_3)

	assert.Equal(t, float64(1), testutil.ToFloat64(gcRunsMetric))
	assert.Equal(t, float64(0), testutil.ToFloat64(foldersRemovedMetric))
	assert.Equal(t, float64(0), testutil.ToFloat64(reclaimedMemoryMetric))

	assert.NoError(t, err)
	gc.assertVersionExists(t, version_1, version_2, version_3)
}

func newMockGarbageCollector() *CSIGarbageCollector {
	return &CSIGarbageCollector{
		logger: logger.NewDTLogger(),
		opts:   dtcsi.CSIOptions{RootDir: rootDir},
		fs:     afero.NewMemMapFs(),
	}
}

func (gc *CSIGarbageCollector) mockUnusedVersions(versions ...string) {
	for _, version := range versions {
		_ = gc.fs.MkdirAll(filepath.Join(versionReferenceBasePath, version), 0770)
	}
}
func (gc *CSIGarbageCollector) mockUsedVersions(versions ...string) {
	for _, version := range versions {
		_ = gc.fs.MkdirAll(filepath.Join(versionReferenceBasePath, version), 0770)
		_, _ = gc.fs.Create(filepath.Join(versionReferenceBasePath, version, "somePodID"))
	}
}

func (gc *CSIGarbageCollector) assertVersionNotExists(t *testing.T, versions ...string) {
	for _, version := range versions {
		exists, err := afero.DirExists(gc.fs, filepath.Join(versionReferenceBasePath, version))
		assert.False(t, exists)
		assert.NoError(t, err)
	}
}

func (gc *CSIGarbageCollector) assertVersionExists(t *testing.T, versions ...string) {
	for _, version := range versions {
		exists, err := afero.DirExists(gc.fs, filepath.Join(versionReferenceBasePath, version))
		assert.True(t, exists)
		assert.NoError(t, err)
	}
}

// This is a very ugly hack, but because you can't Set the value of a Counter metric you have to create new ones to reset them between runs.
func resetMetrics() {
	gcRunsMetric = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "test",
		Subsystem: "csi_driver",
		Name:      "gc_runs",
	})
	foldersRemovedMetric = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "test",
		Subsystem: "csi_driver",
		Name:      "gc_folder_rm",
	})
	reclaimedMemoryMetric = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "test",
		Subsystem: "csi_driver",
		Name:      "gc_memory_reclaimed",
	})
}

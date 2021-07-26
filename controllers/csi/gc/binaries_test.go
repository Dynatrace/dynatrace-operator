package csigc

import (
	"fmt"
	"path/filepath"
	"testing"

	dtcsi "github.com/Dynatrace/dynatrace-operator/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/controllers/csi/storage"
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
	binaryDir = filepath.Join(rootDir, tenantUUID, "bin")
)

func TestBinaryGarbageCollector_getUsedVersions(t *testing.T) {
	gc := NewMockGarbageCollector()
	gc.mockUsedVersions(version_1, version_2, version_3)

	usedVersions, err := gc.db.GetUsedVersions(tenantUUID)
	assert.NoError(t, err)

	assert.NotNil(t, usedVersions)
	assert.Equal(t, len(usedVersions), 3)
	assert.NoError(t, err)
}

func TestBinaryGarbageCollector_succeedsWhenNoVersionPresent(t *testing.T) {
	resetMetrics()
	gc := NewMockGarbageCollector()

	gc.runBinaryGarbageCollection(tenantUUID, version_1)

	assert.Equal(t, float64(1), testutil.ToFloat64(gcRunsMetric))
	assert.Equal(t, float64(0), testutil.ToFloat64(foldersRemovedMetric))
	assert.Equal(t, float64(0), testutil.ToFloat64(reclaimedMemoryMetric))
}

func TestBinaryGarbageCollector_succeedsWhenNoVersionsAvailable(t *testing.T) {
	resetMetrics()
	gc := NewMockGarbageCollector()
	_ = gc.fs.MkdirAll(binaryDir, 0770)

	gc.runBinaryGarbageCollection(tenantUUID, version_1)

	assert.Equal(t, float64(1), testutil.ToFloat64(gcRunsMetric))
	assert.Equal(t, float64(0), testutil.ToFloat64(foldersRemovedMetric))
	assert.Equal(t, float64(0), testutil.ToFloat64(reclaimedMemoryMetric))
}

func TestBinaryGarbageCollector_ignoresLatest(t *testing.T) {
	resetMetrics()
	gc := NewMockGarbageCollector()
	gc.mockUnusedVersions(version_1)

	gc.runBinaryGarbageCollection(tenantUUID, version_1)

	assert.Equal(t, float64(1), testutil.ToFloat64(gcRunsMetric))
	assert.Equal(t, float64(0), testutil.ToFloat64(foldersRemovedMetric))
	assert.Equal(t, float64(0), testutil.ToFloat64(reclaimedMemoryMetric))

	gc.assertVersionExists(t, version_1)
}

func TestBinaryGarbageCollector_removesUnused(t *testing.T) {
	resetMetrics()
	gc := NewMockGarbageCollector()
	gc.mockUnusedVersions(version_1, version_2, version_3)

	gc.runBinaryGarbageCollection(tenantUUID, version_2)

	assert.Equal(t, float64(1), testutil.ToFloat64(gcRunsMetric))
	assert.Equal(t, float64(2), testutil.ToFloat64(foldersRemovedMetric))

	gc.assertVersionNotExists(t, version_1, version_3)
}

func TestBinaryGarbageCollector_ignoresUsed(t *testing.T) {
	resetMetrics()
	gc := NewMockGarbageCollector()
	gc.mockUsedVersions(version_1, version_2, version_3)

	gc.runBinaryGarbageCollection(tenantUUID, version_3)

	assert.Equal(t, float64(1), testutil.ToFloat64(gcRunsMetric))
	assert.Equal(t, float64(0), testutil.ToFloat64(foldersRemovedMetric))
	assert.Equal(t, float64(0), testutil.ToFloat64(reclaimedMemoryMetric))

	gc.assertVersionExists(t, version_1, version_2, version_3)
}

func NewMockGarbageCollector() *CSIGarbageCollector {
	return &CSIGarbageCollector{
		logger: logger.NewDTLogger(),
		opts:   dtcsi.CSIOptions{RootDir: rootDir},
		fs:     afero.NewMemMapFs(),
		db:     storage.FakeMemoryDB(),
		fph:    storage.FilePathHandler{RootDir: rootDir},
	}
}

func (gc *CSIGarbageCollector) mockUnusedVersions(versions ...string) {
	_ = gc.fs.Mkdir(binaryDir, 0770)
	for _, version := range versions {
		_, _ = gc.fs.Create(filepath.Join(binaryDir, version))
	}
}
func (gc *CSIGarbageCollector) mockUsedVersions(versions ...string) {
	_ = gc.fs.Mkdir(binaryDir, 0770)
	for i, version := range versions {
		_, _ = gc.fs.Create(filepath.Join(binaryDir, version))
		_ = gc.db.InsertVolumeInfo(&storage.Volume{ID: fmt.Sprintf("pod%b", i), PodUID: fmt.Sprintf("volume%b", i), Version: version, TenantUUID: tenantUUID})
	}
}

func (gc *CSIGarbageCollector) assertVersionNotExists(t *testing.T, versions ...string) {
	for _, version := range versions {
		exists, err := afero.Exists(gc.fs, filepath.Join(binaryDir, version))
		assert.False(t, exists)
		assert.NoError(t, err)
	}
}

func (gc *CSIGarbageCollector) assertVersionExists(t *testing.T, versions ...string) {
	for _, version := range versions {
		exists, err := afero.Exists(gc.fs, filepath.Join(binaryDir, version))
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

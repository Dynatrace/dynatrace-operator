package csigc

import (
	"fmt"
	"path/filepath"
	"testing"

	dtcsi "github.com/Dynatrace/dynatrace-operator/src/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/csi/metadata"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

const (
	testTenantUUID = "asd12345"
	testVersion1   = "1"
	testVersion2   = "2"
	testVersion3   = "3"
	testRootDir    = "/tmp"
)

var (
	testBinaryDir = filepath.Join(testRootDir, testTenantUUID, "bin")
)

func TestRunBinaryGarbageCollection(t *testing.T) {
	t.Run("succeeds when no version present", func(t *testing.T) {
		resetMetrics()
		gc := NewMockGarbageCollector()

		gc.runBinaryGarbageCollection(pinnedVersionSet{}, testTenantUUID, testVersion1)

		assert.Equal(t, float64(1), testutil.ToFloat64(gcRunsMetric))
		assert.Equal(t, float64(0), testutil.ToFloat64(foldersRemovedMetric))
		assert.Equal(t, float64(0), testutil.ToFloat64(reclaimedMemoryMetric))
	})
	t.Run("succeeds when no version available", func(t *testing.T) {
		resetMetrics()
		gc := NewMockGarbageCollector()
		_ = gc.fs.MkdirAll(testBinaryDir, 0770)

		gc.runBinaryGarbageCollection(pinnedVersionSet{}, testTenantUUID, testVersion1)

		assert.Equal(t, float64(1), testutil.ToFloat64(gcRunsMetric))
		assert.Equal(t, float64(0), testutil.ToFloat64(foldersRemovedMetric))
		assert.Equal(t, float64(0), testutil.ToFloat64(reclaimedMemoryMetric))

	})
	t.Run("ignores latest", func(t *testing.T) {
		resetMetrics()
		gc := NewMockGarbageCollector()
		gc.mockUnusedVersions(testVersion1)

		gc.runBinaryGarbageCollection(pinnedVersionSet{}, testTenantUUID, testVersion1)

		assert.Equal(t, float64(1), testutil.ToFloat64(gcRunsMetric))
		assert.Equal(t, float64(0), testutil.ToFloat64(foldersRemovedMetric))
		assert.Equal(t, float64(0), testutil.ToFloat64(reclaimedMemoryMetric))

		gc.assertVersionExists(t, testVersion1)
	})
	t.Run("remove unused", func(t *testing.T) {
		resetMetrics()
		gc := NewMockGarbageCollector()
		gc.mockUnusedVersions(testVersion1, testVersion2, testVersion3)

		gc.runBinaryGarbageCollection(pinnedVersionSet{}, testTenantUUID, testVersion2)

		assert.Equal(t, float64(1), testutil.ToFloat64(gcRunsMetric))
		assert.Equal(t, float64(2), testutil.ToFloat64(foldersRemovedMetric))

		gc.assertVersionNotExists(t, testVersion1, testVersion3)
	})
	t.Run("ignore used", func(t *testing.T) {
		resetMetrics()
		gc := NewMockGarbageCollector()
		gc.mockUsedVersions(testVersion1, testVersion2, testVersion3)

		gc.runBinaryGarbageCollection(pinnedVersionSet{}, testTenantUUID, testVersion3)

		assert.Equal(t, float64(1), testutil.ToFloat64(gcRunsMetric))
		assert.Equal(t, float64(0), testutil.ToFloat64(foldersRemovedMetric))
		assert.Equal(t, float64(0), testutil.ToFloat64(reclaimedMemoryMetric))

		gc.assertVersionExists(t, testVersion1, testVersion2, testVersion3)
	})
	t.Run("ignore set image", func(t *testing.T) {
		resetMetrics()
		gc := NewMockGarbageCollector()
		gc.mockUsedVersions(testVersion1, testVersion2)

		gc.runBinaryGarbageCollection(pinnedVersionSet{testVersion2: true}, testTenantUUID, testVersion1)

		assert.Equal(t, float64(1), testutil.ToFloat64(gcRunsMetric))
		assert.Equal(t, float64(0), testutil.ToFloat64(foldersRemovedMetric))
		assert.Equal(t, float64(0), testutil.ToFloat64(reclaimedMemoryMetric))

		gc.assertVersionExists(t, testVersion1, testVersion2)
	})
}

func TestBinaryGarbageCollector_getUsedVersions(t *testing.T) {
	gc := NewMockGarbageCollector()
	gc.mockUsedVersions(testVersion1, testVersion2, testVersion3)

	usedVersions, err := gc.db.GetUsedVersions(testTenantUUID)
	assert.NoError(t, err)

	assert.NotNil(t, usedVersions)
	assert.Equal(t, len(usedVersions), 3)
	assert.NoError(t, err)
}

func NewMockGarbageCollector() *CSIGarbageCollector {
	return &CSIGarbageCollector{
		opts: dtcsi.CSIOptions{RootDir: testRootDir},
		fs:   afero.NewMemMapFs(),
		db:   metadata.FakeMemoryDB(),
		path: metadata.PathResolver{RootDir: testRootDir},
	}
}

func (gc *CSIGarbageCollector) mockUnusedVersions(versions ...string) {
	_ = gc.fs.Mkdir(testBinaryDir, 0770)
	for _, version := range versions {
		_, _ = gc.fs.Create(filepath.Join(testBinaryDir, version))
	}
}
func (gc *CSIGarbageCollector) mockUsedVersions(versions ...string) {
	_ = gc.fs.Mkdir(testBinaryDir, 0770)
	for i, version := range versions {
		_, _ = gc.fs.Create(filepath.Join(testBinaryDir, version))
		_ = gc.db.InsertVolume(metadata.NewVolume(fmt.Sprintf("pod%b", i), fmt.Sprintf("volume%b", i), version, testTenantUUID))
	}
}

func (gc *CSIGarbageCollector) assertVersionNotExists(t *testing.T, versions ...string) {
	for _, version := range versions {
		exists, err := afero.Exists(gc.fs, filepath.Join(testBinaryDir, version))
		assert.False(t, exists)
		assert.NoError(t, err)
	}
}

func (gc *CSIGarbageCollector) assertVersionExists(t *testing.T, versions ...string) {
	for _, version := range versions {
		exists, err := afero.Exists(gc.fs, filepath.Join(testBinaryDir, version))
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

package csigc

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/mount"
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

		gc.runBinaryGarbageCollection()

		assert.InDelta(t, 1, testutil.ToFloat64(gcRunsMetric), 0.01)
		assert.InDelta(t, 0, testutil.ToFloat64(foldersRemovedMetric), 0.01)
		assert.InDelta(t, 0, testutil.ToFloat64(reclaimedMemoryMetric), 0.01)
	})
	t.Run("succeeds when no version available", func(t *testing.T) {
		resetMetrics()

		gc := NewMockGarbageCollector()
		_ = gc.fs.MkdirAll(testBinaryDir, 0770)

		gc.runBinaryGarbageCollection()

		assert.InDelta(t, 1, testutil.ToFloat64(gcRunsMetric), 0.01)
		assert.InDelta(t, 0, testutil.ToFloat64(foldersRemovedMetric), 0.01)
		assert.InDelta(t, 0, testutil.ToFloat64(reclaimedMemoryMetric), 0.01)
	})
	t.Run("remove unused", func(t *testing.T) {
		resetMetrics()

		gc := NewMockGarbageCollector()
		gc.mockUnusedVersions(testVersion1, testVersion2, testVersion3)
		gc.time.Set(time.Now().Add(2 * safeRemovalThreshold))

		gc.runBinaryGarbageCollection()

		assert.InDelta(t, 1, testutil.ToFloat64(gcRunsMetric), 0.01)
		assert.InDelta(t, 3, testutil.ToFloat64(foldersRemovedMetric), 0.01)

		gc.assertVersionNotExists(t, testVersion1, testVersion3)
	})
	t.Run("ignore recently deleted", func(t *testing.T) {
		resetMetrics()

		gc := NewMockGarbageCollector()
		gc.mockUnusedVersions(testVersion1, testVersion2, testVersion3)

		gc.runBinaryGarbageCollection()

		assert.InDelta(t, 1, testutil.ToFloat64(gcRunsMetric), 0.01)
		assert.InDelta(t, 0, testutil.ToFloat64(foldersRemovedMetric), 0.01)
		assert.InDelta(t, 0, testutil.ToFloat64(reclaimedMemoryMetric), 0.01)

		gc.assertVersionExists(t, testVersion1, testVersion2, testVersion3)
	})
	t.Run("ignore used", func(t *testing.T) {
		resetMetrics()

		gc := NewMockGarbageCollector()
		gc.mockUsedVersions(t, testVersion1, testVersion2, testVersion3)

		gc.runBinaryGarbageCollection()

		assert.InDelta(t, 1, testutil.ToFloat64(gcRunsMetric), 0.01)
		assert.InDelta(t, 0, testutil.ToFloat64(foldersRemovedMetric), 0.01)
		assert.InDelta(t, 0, testutil.ToFloat64(reclaimedMemoryMetric), 0.01)

		gc.assertVersionExists(t, testVersion1, testVersion2, testVersion3)
	})
}

func NewMockGarbageCollector() *CSIGarbageCollector {
	return &CSIGarbageCollector{
		fs:                    afero.NewMemMapFs(),
		db:                    metadata.FakeMemoryDB(),
		path:                  metadata.PathResolver{RootDir: testRootDir},
		time:                  timeprovider.New(),
		maxUnmountedVolumeAge: defaultMaxUnmountedCsiVolumeAge,
		mounter:               mount.NewFakeMounter([]mount.MountPoint{}),
		isNotMounted:          mockIsNotMounted(map[string]error{}),
	}
}

func (gc *CSIGarbageCollector) mockUnusedVersions(versions ...string) {
	_ = gc.fs.Mkdir(testBinaryDir, 0770)

	gc.isNotMounted = mockIsNotMounted(map[string]error{})
	for _, version := range versions {
		gc.db.(metadata.Access).CreateCodeModule(&metadata.CodeModule{Version: version, Location: filepath.Join(testBinaryDir, version)})
		_, _ = gc.fs.Create(filepath.Join(testBinaryDir, version))
		gc.db.(metadata.Access).DeleteCodeModule(&metadata.CodeModule{Version: version})
	}
}

func (gc *CSIGarbageCollector) mockUsedVersions(t *testing.T, versions ...string) {
	_ = gc.fs.Mkdir(testBinaryDir, 0770)
	for i, version := range versions {
		_, _ = gc.fs.Create(filepath.Join(testBinaryDir, version))
		appMount := metadata.AppMount{
			VolumeMeta:        metadata.VolumeMeta{ID: fmt.Sprintf("volume%b", i), PodName: fmt.Sprintf("pod%b", i)},
			VolumeMetaID:      fmt.Sprintf("volume%b", i),
			CodeModuleVersion: version,
			MountAttempts:     0,
		}
		err := gc.db.(metadata.Access).CreateAppMount(&appMount)
		require.NoError(t, err)

		gc.db.(metadata.Access).CreateCodeModule(&metadata.CodeModule{Version: version, Location: filepath.Join(testBinaryDir, version)})
	}
}

func (gc *CSIGarbageCollector) assertVersionNotExists(t *testing.T, versions ...string) {
	for _, version := range versions {
		exists, err := afero.Exists(gc.fs, filepath.Join(testBinaryDir, version))
		assert.False(t, exists)
		require.NoError(t, err)
	}
}

func (gc *CSIGarbageCollector) assertVersionExists(t *testing.T, versions ...string) {
	for _, version := range versions {
		exists, err := afero.Exists(gc.fs, filepath.Join(testBinaryDir, version))
		assert.True(t, exists)
		require.NoError(t, err)
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

package csigc

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var (
	testLogPath    = filepath.Join(testRootDir, testTenantUUID, "run")
	testTechnology = "go"
)

func TestLogGarbageCollector_noErrorWithoutLogs(t *testing.T) {
	gc := NewMockGarbageCollector()

	_ = gc.fs.MkdirAll(testLogPath, 0770)
	logs, err := gc.getLogFileInfo(testTenantUUID)

	assert.NoError(t, err)
	assert.Equal(t, &logFileInfo{
		UnusedVolumeIDs: []os.FileInfo(nil),
		NumberOfFiles:   0,
		OverallSize:     0,
	}, logs)
}

func TestLogGarbageCollector_emptyLogFileInfoWithNoUnmountedLogs(t *testing.T) {
	gc := NewMockGarbageCollector()
	gc.mockMountedVolumeIDPath(testVersion1)

	logs, err := gc.getLogFileInfo(testTenantUUID)

	assert.NoError(t, err)
	assert.Equal(t, &logFileInfo{
		UnusedVolumeIDs: []os.FileInfo(nil),
		NumberOfFiles:   0,
		OverallSize:     0,
	}, logs)
}

func TestLogGarbageCollector_logFileInfo_JustVolumeID_WithUnmountedLogs(t *testing.T) {
	gc := NewMockGarbageCollector()

	gc.mockUnmountedVolumeIDPath(testVersion1)

	logs, err := gc.getLogFileInfo(testTenantUUID)

	assert.NoError(t, err)
	assert.Equal(t, int64(0), logs.NumberOfFiles)
	assert.Equal(t, int64(0), logs.OverallSize)
	assert.Equal(t, testVersion1, logs.UnusedVolumeIDs[0].Name())
}

func TestLogGarbageCollector_logFileInfo_SingleVolumeID_WithUnmountedLogs(t *testing.T) {
	gc := NewMockGarbageCollector()

	gc.mockUnmountedVolumeIDPath(testVersion1)
	gc.mockLogsInPodFolders(5, testVersion1)

	logs, err := gc.getLogFileInfo(testTenantUUID)

	assert.NoError(t, err)
	assert.Equal(t, int64(5), logs.NumberOfFiles)
	assert.Equal(t, int64(0), logs.OverallSize)
	assert.Equal(t, testVersion1, logs.UnusedVolumeIDs[0].Name())
}

func TestLogGarbageCollector_logFileInfo_MultipleVolumeIDs_WithUnmountedLogs(t *testing.T) {
	gc := NewMockGarbageCollector()

	gc.mockUnmountedVolumeIDPath(testVersion1, testVersion2, testVersion3)
	gc.mockLogsInPodFolders(5, testVersion1, testVersion2)

	logs, err := gc.getLogFileInfo(testTenantUUID)

	assert.NoError(t, err)
	assert.Equal(t, int64(10), logs.NumberOfFiles)
	assert.Equal(t, int64(0), logs.OverallSize)
	assert.Equal(t, testVersion1, logs.UnusedVolumeIDs[0].Name())
	assert.Equal(t, testVersion2, logs.UnusedVolumeIDs[1].Name())
	assert.Equal(t, testVersion3, logs.UnusedVolumeIDs[2].Name())
}

func TestLogGarbageCollector_logFileInfo_MultipleVolumeIDs_WithUnmountedAndMountedLogs(t *testing.T) {
	gc := NewMockGarbageCollector()

	gc.mockMountedVolumeIDPath(testVersion3)
	gc.mockUnmountedVolumeIDPath(testVersion1, testVersion2)
	gc.mockLogsInPodFolders(5, testVersion1, testVersion2)

	logs, err := gc.getLogFileInfo(testTenantUUID)

	assert.NoError(t, err)
	assert.Equal(t, int64(10), logs.NumberOfFiles)
	assert.Equal(t, int64(0), logs.OverallSize)
	assert.Equal(t, testVersion1, logs.UnusedVolumeIDs[0].Name())
	assert.Equal(t, testVersion2, logs.UnusedVolumeIDs[1].Name())
	assert.Equal(t, 2, len(logs.UnusedVolumeIDs))
}

func TestLogGarbageCollector_modificationDateOlderThanTwoWeeks(t *testing.T) {
	t.Run("is false for current timestamp", func(t *testing.T) {
		isOlder := isOlderThanTwoWeeks(time.Now())

		assert.False(t, isOlder)
	})

	t.Run("is true for timestamp 14 days in past", func(t *testing.T) {
		isOlder := isOlderThanTwoWeeks(time.Now().AddDate(0, 0, -15))

		assert.True(t, isOlder)
	})
}

func TestLogGarbageCollector_cleanUpSuccessful(t *testing.T) {
	gc := NewMockGarbageCollector()

	gc.mockUnmountedVolumeIDPath(testVersion1, testVersion2)
	gc.mockLogsInPodFolders(5, testVersion1, testVersion2)

	logs, err := gc.getLogFileInfo(testTenantUUID)
	assert.NoError(t, err)
	assert.NotNil(t, logs)

	older := isOlderThanTwoWeeks(logs.UnusedVolumeIDs[0].ModTime())
	assert.True(t, older)

	gc.tryRemoveLogFolders(logs.UnusedVolumeIDs, testTenantUUID)
	assert.NoDirExists(t, filepath.Join(testLogPath, logs.UnusedVolumeIDs[0].Name()))
}

func TestLogGarbageCollector_removeLogsNecessary_filesGetDeleted(t *testing.T) {
	gc := NewMockGarbageCollector()

	gc.mockUnmountedVolumeIDPath(testVersion1, testVersion2)
	gc.mockLogsInPodFolders(5, testVersion1, testVersion2)

	logs, err := gc.getLogFileInfo(testTenantUUID)
	assert.NoError(t, err)
	assert.NotNil(t, logs)

	gc.removeLogsIfNecessary(logs, int64(0), int64(1), testTenantUUID)
	newLogs, err := gc.getLogFileInfo(testTenantUUID)
	assert.NoError(t, err)
	assert.NotEqual(t, newLogs, logs)
	assert.Equal(t, newLogs.NumberOfFiles, int64(0))
}

func TestLogGarbageCollector_removeLogsNecessary_tooLessFiles(t *testing.T) {
	gc := NewMockGarbageCollector()

	gc.mockUnmountedVolumeIDPath(testVersion1, testVersion2)
	gc.mockLogsInPodFolders(5, testVersion1, testVersion2)

	logs, err := gc.getLogFileInfo(testTenantUUID)
	assert.NoError(t, err)
	assert.NotNil(t, logs)

	gc.removeLogsIfNecessary(logs, int64(0), int64(11), testTenantUUID)
	newLogs, err := gc.getLogFileInfo(testTenantUUID)
	assert.NoError(t, err)
	assert.Equal(t, newLogs, logs)
	assert.Equal(t, newLogs.NumberOfFiles, int64(10))
}

func TestLogGarbageCollector_removeLogsNecessary_FileSizeTooSmall(t *testing.T) {
	gc := NewMockGarbageCollector()

	gc.mockUnmountedVolumeIDPath(testVersion1, testVersion2)
	gc.mockLogsInPodFolders(5, testVersion1, testVersion2)

	logs, err := gc.getLogFileInfo(testTenantUUID)
	assert.NoError(t, err)
	assert.NotNil(t, logs)

	gc.removeLogsIfNecessary(logs, int64(10), int64(11), testTenantUUID)
	newLogs, err := gc.getLogFileInfo(testTenantUUID)
	assert.NoError(t, err)
	assert.Equal(t, newLogs, logs)
	assert.Equal(t, newLogs.NumberOfFiles, int64(10))
}

func (gc *CSIGarbageCollector) mockMountedVolumeIDPath(volumeIDs ...string) {
	for _, volumeID := range volumeIDs {
		_ = gc.fs.MkdirAll(filepath.Join(testLogPath, volumeID, "mapped", "something"), os.ModePerm)
	}
}

func (gc *CSIGarbageCollector) mockUnmountedVolumeIDPath(volumeIDs ...string) {
	for _, volumeID := range volumeIDs {
		_ = gc.fs.MkdirAll(filepath.Join(testLogPath, volumeID, "mapped"), os.ModePerm)
	}
}

func (gc *CSIGarbageCollector) mockLogsInPodFolders(nrOfLogFiles int, volumeIDs ...string) {
	for _, volumeID := range volumeIDs {
		technologyLogPath := filepath.Join(testLogPath, volumeID, "var", "log", testTechnology)
		_ = gc.fs.Mkdir(filepath.Join(technologyLogPath), 0770)
		for i := 0; i < nrOfLogFiles; i++ {
			_, _ = gc.fs.Create(filepath.Join(technologyLogPath, "logfile"+strconv.Itoa(i)))
		}
	}
}

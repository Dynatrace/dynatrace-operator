package csigc

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

var (
	logPath    = filepath.Join(rootDir, tenantUUID, "run")
	technology = "go"
)

func TestLogGarbageCollector_noErrorWithoutLogs(t *testing.T) {
	gc := NewMockGarbageCollector()

	_ = gc.fs.MkdirAll(logPath, 0770)
	logs, err := gc.getLogFileInfo(tenantUUID, &afero.Afero{Fs: gc.fs})

	assert.NoError(t, err)
	assert.Equal(t, &logFileInfo{
		UnusedVolumeIDs: []os.FileInfo(nil),
		NumberOfFiles:   0,
		OverallSize:     0,
	}, logs)
}

func TestLogGarbageCollector_emptyLogFileInfoWithNoUnmountedLogs(t *testing.T) {
	gc := NewMockGarbageCollector()

	_ = gc.fs.MkdirAll(logPath, 0770)
	gc.mockMountedVolumeIDPath(version_1)

	logs, err := gc.getLogFileInfo(tenantUUID, &afero.Afero{Fs: gc.fs})

	assert.NoError(t, err)
	assert.Equal(t, &logFileInfo{
		UnusedVolumeIDs: []os.FileInfo(nil),
		NumberOfFiles:   0,
		OverallSize:     0,
	}, logs)
}

func TestLogGarbageCollector_logFileInfo_JustVolumeID_WithUnmountedLogs(t *testing.T) {
	gc := NewMockGarbageCollector()

	_ = gc.fs.MkdirAll(logPath, 0770)
	gc.mockUnmountedVolumeIDPath(version_1)

	logs, err := gc.getLogFileInfo(tenantUUID, &afero.Afero{Fs: gc.fs})

	assert.NoError(t, err)
	assert.Equal(t, int64(0), logs.NumberOfFiles)
	assert.Equal(t, int64(0), logs.OverallSize)
	assert.Equal(t, version_1, logs.UnusedVolumeIDs[0].Name())
}

func TestLogGarbageCollector_logFileInfo_SingleVolumeID_WithUnmountedLogs(t *testing.T) {
	gc := NewMockGarbageCollector()

	_ = gc.fs.MkdirAll(logPath, 0770)
	gc.mockUnmountedVolumeIDPath(version_1)
	gc.mockLogsInPodFolders(5, version_1)

	logs, err := gc.getLogFileInfo(tenantUUID, &afero.Afero{Fs: gc.fs})

	assert.NoError(t, err)
	assert.Equal(t, int64(5), logs.NumberOfFiles)
	assert.Equal(t, int64(0), logs.OverallSize)
	assert.Equal(t, version_1, logs.UnusedVolumeIDs[0].Name())
}

func TestLogGarbageCollector_logFileInfo_MultipleVolumeIDs_WithUnmountedLogs(t *testing.T) {
	gc := NewMockGarbageCollector()

	_ = gc.fs.MkdirAll(logPath, 0770)
	gc.mockUnmountedVolumeIDPath(version_1, version_2, version_3)
	gc.mockLogsInPodFolders(5, version_1, version_2)

	logs, err := gc.getLogFileInfo(tenantUUID, &afero.Afero{Fs: gc.fs})

	assert.NoError(t, err)
	assert.Equal(t, int64(10), logs.NumberOfFiles)
	assert.Equal(t, int64(0), logs.OverallSize)
	assert.Equal(t, version_1, logs.UnusedVolumeIDs[0].Name())
	assert.Equal(t, version_2, logs.UnusedVolumeIDs[1].Name())
	assert.Equal(t, version_3, logs.UnusedVolumeIDs[2].Name())
}

func TestLogGarbageCollector_logFileInfo_MultipleVolumeIDs_WithUnmountedAndMountedLogs(t *testing.T) {
	gc := NewMockGarbageCollector()

	_ = gc.fs.MkdirAll(logPath, 0770)
	gc.mockMountedVolumeIDPath(version_3)
	gc.mockUnmountedVolumeIDPath(version_1, version_2)
	gc.mockLogsInPodFolders(5, version_1, version_2)

	logs, err := gc.getLogFileInfo(tenantUUID, &afero.Afero{Fs: gc.fs})

	assert.NoError(t, err)
	assert.Equal(t, int64(10), logs.NumberOfFiles)
	assert.Equal(t, int64(0), logs.OverallSize)
	assert.Equal(t, version_1, logs.UnusedVolumeIDs[0].Name())
	assert.Equal(t, version_2, logs.UnusedVolumeIDs[1].Name())
	assert.Equal(t, 2, len(logs.UnusedVolumeIDs))
}

func TestLogGarbageCollector_modificationDateOlderThanTwoWeeks(t *testing.T) {
	gc := NewMockGarbageCollector()

	_ = gc.fs.MkdirAll(logPath, 0770)
	gc.mockUnmountedVolumeIDPath(version_1, version_2)
	gc.mockLogsInPodFolders(5, version_1, version_2)

	logs, err := gc.getLogFileInfo(tenantUUID, &afero.Afero{Fs: gc.fs})
	assert.NoError(t, err)
	assert.NotNil(t, logs)

	older := isOlderThanTwoWeeks(logs.UnusedVolumeIDs[0].ModTime())
	assert.True(t, older)
}

func TestLogGarbageCollector_cleanUpSuccessful(t *testing.T) {
	gc := NewMockGarbageCollector()

	_ = gc.fs.MkdirAll(logPath, 0770)
	gc.mockUnmountedVolumeIDPath(version_1, version_2)
	gc.mockLogsInPodFolders(5, version_1, version_2)

	logs, err := gc.getLogFileInfo(tenantUUID, &afero.Afero{Fs: gc.fs})
	assert.NoError(t, err)
	assert.NotNil(t, logs)

	older := isOlderThanTwoWeeks(logs.UnusedVolumeIDs[0].ModTime())
	assert.True(t, older)

	gc.tryRemoveLogFolders(logs.UnusedVolumeIDs, tenantUUID, &afero.Afero{Fs: gc.fs})
	assert.NoDirExists(t, filepath.Join(logPath, logs.UnusedVolumeIDs[0].Name()))
}

func (gc *CSIGarbageCollector) mockMountedVolumeIDPath(volumeIDs ...string) {
	for _, volumeID := range volumeIDs {
		_ = gc.fs.MkdirAll(filepath.Join(logPath, volumeID, "mapped", "something"), os.ModePerm)
	}
}

func (gc *CSIGarbageCollector) mockUnmountedVolumeIDPath(volumeIDs ...string) {
	for _, volumeID := range volumeIDs {
		_ = gc.fs.MkdirAll(filepath.Join(logPath, volumeID, "mapped"), os.ModePerm)
	}
}

func (gc *CSIGarbageCollector) mockLogsInPodFolders(nrOfLogFiles int, volumeIDs ...string) {
	for _, volumeID := range volumeIDs {
		technologyLogPath := filepath.Join(logPath, volumeID, "var", "log", technology)
		_ = gc.fs.Mkdir(filepath.Join(technologyLogPath), 0770)
		for i := 0; i < nrOfLogFiles; i++ {
			_, _ = gc.fs.Create(filepath.Join(technologyLogPath, "logfile"+strconv.Itoa(i)))
		}
	}
}

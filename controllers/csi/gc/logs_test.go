package csigc

import (
	"path/filepath"
	"strconv"
	"testing"

	dtcsi "github.com/Dynatrace/dynatrace-operator/controllers/csi"
	"github.com/stretchr/testify/assert"
)

var (
	logPath    = filepath.Join(rootDir, dtcsi.DataPath, tenantUUID, dtcsi.LogDir)
	technology = "go"
)

func TestLogGarbageCollector_succeedsWhenNoFilesAreGiven(t *testing.T) {
	gc := NewMockGarbageCollector()

	_ = gc.fs.MkdirAll(versionReferenceBasePath, 0770)
	_ = gc.fs.MkdirAll(logPath, 0770)

	versionReferences, err := gc.getVersionReferences(tenantUUID)
	assert.NoError(t, err)

	err = gc.runLogGarbageCollection(versionReferences, tenantUUID)
	assert.NoError(t, err)
}

func TestLogGarbageCollector_succeedsUsedFilesFound(t *testing.T) {
	gc := NewMockGarbageCollector()

	_ = gc.fs.MkdirAll(versionReferenceBasePath, 0770)
	_ = gc.fs.MkdirAll(logPath, 0770)
	gc.MockUsedVersions(version_1, version_2, version_3)

	versionReferences, err := gc.getVersionReferences(tenantUUID)
	assert.NoError(t, err)

	err = gc.runLogGarbageCollection(versionReferences, tenantUUID)
	assert.NoError(t, err)
}

func TestLogGarbageCollector_assertNumberOfLogsFilesAreEqual(t *testing.T) {
	gc := NewMockGarbageCollector()

	_ = gc.fs.MkdirAll(versionReferenceBasePath, 0770)
	_ = gc.fs.MkdirAll(logPath, 0770)

	versionReferences, err := gc.getVersionReferences(tenantUUID)
	assert.NoError(t, err)

	gc.mockLogPodFolders(version_1, version_2)
	gc.mockLogsInPodFolders(5, version_1, version_2)

	err = gc.runLogGarbageCollection(versionReferences, tenantUUID)
	assert.NoError(t, err)
}

func (gc *CSIGarbageCollector) mockLogPodFolders(podIDs ...string) {
	for _, podID := range podIDs {
		_, _ = gc.fs.Create(filepath.Join(logPath, podID))
	}
}

func (gc *CSIGarbageCollector) mockLogsInPodFolders(nrOfLogFiles int, podIDs ...string) {
	for _, podID := range podIDs {
		_ = gc.fs.Mkdir(filepath.Join(logPath, podID, technology), 0770)
		for i := 0; i < nrOfLogFiles; i++ {
			_, _ = gc.fs.Create(filepath.Join(logPath, podID, technology, "logfile"+strconv.Itoa(i)))
		}
	}
}

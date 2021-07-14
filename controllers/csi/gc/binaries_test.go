package csigc

import (
	"path/filepath"
	"testing"

	dtcsi "github.com/Dynatrace/dynatrace-operator/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/logger"
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

func TestBinaryGarbageCollector_versionReferenceGenerationSuccess(t *testing.T) {
	gc := NewMockGarbageCollector()
	gc.mockUnusedVersions(version_1, version_2, version_3)

	versionReferences, err := gc.getVersionReferences(tenantUUID)
	assert.NoError(t, err)

	assert.NotNil(t, versionReferences)
	assert.NoError(t, err)
}

func TestBinaryGarbageCollector_ignoresLatest(t *testing.T) {
	gc := NewMockGarbageCollector()
	gc.mockUnusedVersions(version_1)

	gc.runBinaryGarbageCollection(tenantUUID, version_1)

	gc.assertVersionExists(t, version_1)
}

func TestBinaryGarbageCollector_removesUnused(t *testing.T) {
	gc := NewMockGarbageCollector()
	gc.mockUnusedVersions(version_1, version_2, version_3)

	gc.runBinaryGarbageCollection(tenantUUID, version_2)

	gc.assertVersionNotExists(t, version_1, version_3)
}

func TestBinaryGarbageCollector_ignoresUsed(t *testing.T) {
	gc := NewMockGarbageCollector()
	gc.mockUsedVersions(version_1, version_2, version_3)

	gc.runBinaryGarbageCollection(tenantUUID, version_3)

	gc.assertVersionExists(t, version_1, version_2, version_3)
}

func NewMockGarbageCollector() *CSIGarbageCollector {
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

package csigc

import (
	"path/filepath"
	"testing"

	dtcsi "github.com/Dynatrace/dynatrace-operator/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/controllers/dynakube"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/Dynatrace/dynatrace-operator/scheme/fake"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

const envID = "asd12345"

func newMockGarbageCollector() (CSIGarbageCollector, string) {
	clt := fake.NewClient()
	dtcMock := &dtclient.MockDynatraceClient{}
	log := logger.NewDTLogger()
	fs := afero.NewMemMapFs()
	opts := dtcsi.CSIOptions{
		GCInterval: 1,
		RootDir:    "/tmp",
		Endpoint:   "endpoint",
		NodeID:     "nodeID123",
	}

	versionReferenceBasePath := filepath.Join(opts.RootDir, dtcsi.DataPath, envID, dtcsi.GarbageCollectionPath)
	_ = fs.MkdirAll(versionReferenceBasePath, 0770)

	return CSIGarbageCollector{
		client:       clt,
		dtcBuildFunc: dynakube.StaticDynatraceClient(dtcMock),
		logger:       log,
		opts:         opts,
		fs:           fs,
	}, versionReferenceBasePath

}

func TestBinaryGarbageCollector_BinaryGarbageCollection(t *testing.T) {
	gc, versionReferenceBasePath := newMockGarbageCollector()

	oldVersion := "1.100.0"
	latestVersion := "1.101.0"

	if err := gc.fs.MkdirAll(filepath.Join(versionReferenceBasePath, oldVersion), 0770); err != nil {
		assert.NoError(t, err)
	}

	t.Run(`only latest version available`, func(t *testing.T) {
		_ = gc.fs.MkdirAll(filepath.Join(versionReferenceBasePath, latestVersion), 0770)

		err := gc.runBinaryGarbageCollection(envID, latestVersion)
		assert.NoError(t, err)

		exists, err := afero.DirExists(gc.fs, filepath.Join(versionReferenceBasePath, latestVersion))
		assert.True(t, exists)
		assert.NoError(t, err)
	})

	t.Run(`garbage collector removes unused version`, func(t *testing.T) {
		_ = gc.fs.MkdirAll(filepath.Join(versionReferenceBasePath, latestVersion), 0770)
		_ = gc.fs.MkdirAll(filepath.Join(versionReferenceBasePath, oldVersion), 0770)

		err := gc.runBinaryGarbageCollection(envID, latestVersion)
		assert.NoError(t, err)

		latestExists, err := afero.DirExists(gc.fs, filepath.Join(versionReferenceBasePath, latestVersion))
		assert.True(t, latestExists)
		assert.NoError(t, err)

		oldExists, err := afero.DirExists(gc.fs, filepath.Join(versionReferenceBasePath, oldVersion))
		assert.False(t, oldExists)
		assert.NoError(t, err)
	})

	t.Run(`garbage collector not removes used version`, func(t *testing.T) {
		_ = gc.fs.MkdirAll(filepath.Join(versionReferenceBasePath, latestVersion), 0770)
		_ = gc.fs.MkdirAll(filepath.Join(versionReferenceBasePath, oldVersion), 0770)
		_, _ = gc.fs.Create(filepath.Join(versionReferenceBasePath, oldVersion, "somePodID"))

		err := gc.runBinaryGarbageCollection(envID, latestVersion)
		assert.NoError(t, err)

		latestExists, err := afero.DirExists(gc.fs, filepath.Join(versionReferenceBasePath, latestVersion))
		assert.True(t, latestExists)
		assert.NoError(t, err)

		oldExists, err := afero.DirExists(gc.fs, filepath.Join(versionReferenceBasePath, oldVersion))
		assert.True(t, oldExists)
		assert.NoError(t, err)
	})

	t.Run(`garbage collecting no version`, func(t *testing.T) {
		err := gc.runBinaryGarbageCollection(envID, latestVersion)
		assert.NoError(t, err)
	})
}

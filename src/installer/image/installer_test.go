package image

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/controllers/csi/metadata"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsAlreadyDownloaded(t *testing.T) {
	imageDigest := "test"
	pathResolver := metadata.PathResolver{}

	t.Run(`returns early if path doesn't exist`, func(t *testing.T) {
		installer := ImageInstaller{
			fs: afero.NewMemMapFs(),
			props: &Properties{
				PathResolver: pathResolver,
			},
		}
		isDownloaded, err := installer.isAlreadyDownloaded(imageDigest)
		require.NoError(t, err)
		assert.False(t, isDownloaded)
	})
	t.Run(`checks metadata if path already exists, no entry => previous download failed`, func(t *testing.T) {
		installer := ImageInstaller{
			fs: testFileSystemWithSharedDirPresent(pathResolver, imageDigest),
			props: &Properties{
				PathResolver: pathResolver,
				Metadata: metadata.FakeMemoryDB(),
			},
		}
		isDownloaded, err := installer.isAlreadyDownloaded(imageDigest)
		require.NoError(t, err)
		assert.False(t, isDownloaded)
	})
	t.Run(`checks metadata if path already exists, entry => previous download succeeded`, func(t *testing.T) {
		installer := ImageInstaller{
			fs: testFileSystemWithSharedDirPresent(pathResolver, imageDigest),
			props: &Properties{
				PathResolver: pathResolver,
				Metadata: testMetadataWithImageDigestPresent(imageDigest),
			},
		}
		isDownloaded, err := installer.isAlreadyDownloaded(imageDigest)
		require.NoError(t, err)
		assert.True(t, isDownloaded)
	})
	t.Run(`fail db`, func(t *testing.T) {
		installer := ImageInstaller{
			fs: testFileSystemWithSharedDirPresent(pathResolver, imageDigest),
			props: &Properties{
				PathResolver: pathResolver,
				Metadata: &metadata.FakeFailDB{},
			},
		}
		isDownloaded, err := installer.isAlreadyDownloaded(imageDigest)
		require.Error(t, err)
		assert.False(t, isDownloaded)
	})
}

func testFileSystemWithSharedDirPresent(pathResolver metadata.PathResolver, imageDigest string) afero.Fs {
	fs := afero.NewMemMapFs()
	fs.MkdirAll(pathResolver.AgentSharedBinaryDirForImage(imageDigest), 0777)
	return fs
}

func testMetadataWithImageDigestPresent(imageDigest string) metadata.Access {
	db := metadata.FakeMemoryDB()
	db.InsertDynakube(&metadata.Dynakube{
		Name: "test",
		TenantUUID: "test",
		LatestVersion: "test",
		ImageDigest: imageDigest,
	})
	return db
}

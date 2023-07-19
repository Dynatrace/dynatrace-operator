package image

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/controllers/csi/metadata"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestIsAlreadyPresent(t *testing.T) {
	imageDigest := "test"
	pathResolver := metadata.PathResolver{}

	t.Run("returns early if path doesn't exist", func(t *testing.T) {
		installer := Installer{
			fs: afero.NewMemMapFs(),
			props: &Properties{
				PathResolver: pathResolver,
			},
		}
		isDownloaded := installer.isAlreadyPresent(pathResolver.AgentSharedBinaryDirForAgent(imageDigest))
		assert.False(t, isDownloaded)
	})
	t.Run("returns true if path present", func(t *testing.T) {
		installer := Installer{
			fs: testFileSystemWithSharedDirPresent(pathResolver, imageDigest),
			props: &Properties{
				PathResolver: pathResolver,
			},
		}
		isDownloaded := installer.isAlreadyPresent(pathResolver.AgentSharedBinaryDirForAgent(imageDigest))
		assert.True(t, isDownloaded)
	})
}

func testFileSystemWithSharedDirPresent(pathResolver metadata.PathResolver, imageDigest string) afero.Fs {
	fs := afero.NewMemMapFs()
	fs.MkdirAll(pathResolver.AgentSharedBinaryDirForAgent(imageDigest), 0777)
	return fs
}

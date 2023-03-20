package image

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/controllers/csi/metadata"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestIsAlreadyDownloaded(t *testing.T) {
	imageDigest := "test"
	imageWithDigest := "quay.io/somerepo/codemod@sha256:7ece13a07a20c77a31cc36906a10ebc90bd47970905ee61e8ed491b7f4c5d62f"
	pathResolver := metadata.PathResolver{}

	t.Run(`returns early if path doesn't exist`, func(t *testing.T) {
		installer := ImageInstaller{
			fs: afero.NewMemMapFs(),
			props: &Properties{
				PathResolver: pathResolver,
			},
		}
		isDownloaded := installer.isAlreadyDownloaded(imageDigest)
		assert.False(t, isDownloaded)
	})
	t.Run(`returns true if path present`, func(t *testing.T) {
		installer := ImageInstaller{
			fs: testFileSystemWithSharedDirPresent(pathResolver, imageDigest),
			props: &Properties{
				PathResolver: pathResolver,
			},
		}
		isDownloaded := installer.isAlreadyDownloaded(imageDigest)
		assert.True(t, isDownloaded)
	})
	t.Run(`returns proper digest`, func(t *testing.T) {
		digest := getImageDigestFromImageName(imageWithDigest)
		assert.NotEmpty(t, digest)
		assert.Equal(t, "7ece13a07a20c77a31cc36906a10ebc90bd47970905ee61e8ed491b7f4c5d62f", digest.Encoded())
	})
}

func testFileSystemWithSharedDirPresent(pathResolver metadata.PathResolver, imageDigest string) afero.Fs {
	fs := afero.NewMemMapFs()
	fs.MkdirAll(pathResolver.AgentSharedBinaryDirForImage(imageDigest), 0777)
	return fs
}

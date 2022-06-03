package standalone

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateConfFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	runner := Runner{
		fs: fs,
	}
	t.Run(`create file`, func(t *testing.T) {
		path := "test"

		err := runner.createConfFile(path, "test")

		require.NoError(t, err)

		file, err := fs.Open(path)
		require.NoError(t, err)
		content, err := ioutil.ReadAll(file)
		require.NoError(t, err)
		assert.Contains(t, string(content), "test")

	})
	t.Run(`create nested file`, func(t *testing.T) {
		path := filepath.Join("dir1", "dir2", "test")

		err := runner.createConfFile(path, "test")

		require.NoError(t, err)

		file, err := fs.Open(path)
		require.NoError(t, err)
		content, err := ioutil.ReadAll(file)
		require.NoError(t, err)
		assert.Contains(t, string(content), "test")

	})
}

func TestCurlOptions(t *testing.T) {
	filesystem := afero.NewMemMapFs()
	runner := Runner{
		config: &SecretConfig{InitialConnectRetry: 30},
		fs:     filesystem,
	}

	assert.Equal(t, "initialConnectRetryMs 30\n", runner.getCurlOptionsContent())

	err := runner.createCurlOptionsFile()

	assert.NoError(t, err)

	exists, err := afero.Exists(filesystem, "mnt/share/curl_options.conf")

	assert.NoError(t, err)
	assert.True(t, exists)

	content, err := afero.ReadFile(filesystem, "mnt/share/curl_options.conf")

	assert.NoError(t, err)
	assert.Equal(t, runner.getCurlOptionsContent(), string(content))
}

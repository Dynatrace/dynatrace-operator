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

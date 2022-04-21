package installer

import (
	"encoding/base64"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testZip  = `UEsDBAoAAAAAAKh0p1JsLSFnGQAAABkAAAAIABwAdGVzdC50eHRVVAkAA3w0lWATB55gdXgLAAEE6AMAAAToAwAAeW91IGZvdW5kIHRoZSBlYXN0ZXIgZWdnClBLAwQKAAAAAADAOa5SAAAAAAAAAAAAAAAABQAcAHRlc3QvVVQJAAMXB55gHQeeYHV4CwABBOgDAAAE6AMAAFBLAwQKAAAAAACodKdSbC0hZxkAAAAZAAAADQAcAHRlc3QvdGVzdC50eHRVVAkAA3w0lWATB55gdXgLAAEE6AMAAAToAwAAeW91IGZvdW5kIHRoZSBlYXN0ZXIgZWdnClBLAwQKAAAAAADCOa5SAAAAAAAAAAAAAAAACgAcAHRlc3QvdGVzdC9VVAkAAxwHnmAgB55gdXgLAAEE6AMAAAToAwAAUEsDBAoAAAAAAKh0p1JsLSFnGQAAABkAAAASABwAdGVzdC90ZXN0L3Rlc3QudHh0VVQJAAN8NJVgHAeeYHV4CwABBOgDAAAE6AMAAHlvdSBmb3VuZCB0aGUgZWFzdGVyIGVnZwpQSwMECgAAAAAA2zquUgAAAAAAAAAAAAAAAAYAHABhZ2VudC9VVAkAAy4JnmAxCZ5gdXgLAAEE6AMAAAToAwAAUEsDBAoAAAAAAOI6rlIAAAAAAAAAAAAAAAALABwAYWdlbnQvY29uZi9VVAkAAzgJnmA+CZ5gdXgLAAEE6AMAAAToAwAAUEsDBAoAAAAAAKh0p1JsLSFnGQAAABkAAAATABwAYWdlbnQvY29uZi90ZXN0LnR4dFVUCQADfDSVYDgJnmB1eAsAAQToAwAABOgDAAB5b3UgZm91bmQgdGhlIGVhc3RlciBlZ2cKUEsBAh4DCgAAAAAAqHSnUmwtIWcZAAAAGQAAAAgAGAAAAAAAAQAAAKSBAAAAAHRlc3QudHh0VVQFAAN8NJVgdXgLAAEE6AMAAAToAwAAUEsBAh4DCgAAAAAAwDmuUgAAAAAAAAAAAAAAAAUAGAAAAAAAAAAQAO1BWwAAAHRlc3QvVVQFAAMXB55gdXgLAAEE6AMAAAToAwAAUEsBAh4DCgAAAAAAqHSnUmwtIWcZAAAAGQAAAA0AGAAAAAAAAQAAAKSBmgAAAHRlc3QvdGVzdC50eHRVVAUAA3w0lWB1eAsAAQToAwAABOgDAABQSwECHgMKAAAAAADCOa5SAAAAAAAAAAAAAAAACgAYAAAAAAAAABAA7UH6AAAAdGVzdC90ZXN0L1VUBQADHAeeYHV4CwABBOgDAAAE6AMAAFBLAQIeAwoAAAAAAKh0p1JsLSFnGQAAABkAAAASABgAAAAAAAEAAACkgT4BAAB0ZXN0L3Rlc3QvdGVzdC50eHRVVAUAA3w0lWB1eAsAAQToAwAABOgDAABQSwECHgMKAAAAAADbOq5SAAAAAAAAAAAAAAAABgAYAAAAAAAAABAA7UGjAQAAYWdlbnQvVVQFAAMuCZ5gdXgLAAEE6AMAAAToAwAAUEsBAh4DCgAAAAAA4jquUgAAAAAAAAAAAAAAAAsAGAAAAAAAAAAQAO1B4wEAAGFnZW50L2NvbmYvVVQFAAM4CZ5gdXgLAAEE6AMAAAToAwAAUEsBAh4DCgAAAAAAqHSnUmwtIWcZAAAAGQAAABMAGAAAAAAAAQAAAKSBKAIAAGFnZW50L2NvbmYvdGVzdC50eHRVVAUAA3w0lWB1eAsAAQToAwAABOgDAABQSwUGAAAAAAgACACKAgAAjgIAAAAA`
	testGzip = `H4sIAJwcYWIAA+2WzUrDQBRGZ52nmCdI753fZFGwWKUrKWIXgpvQDhpsLGRSybJ7dz6qT+AkWEMqLqSOUjKHwA2TwNxJcr7JPKtnJluZclQZWxEvAICWkhLd0lX4AIFRFIwDQ+TugqvAFKG1n3b6bG2Vla6VIiuXZr2OrXle2vzxy33T26vJzfXk/OJuuimy/IkurCltb5EO+llPBEzofb4aI+iEpygEi9zIthkRHBIECTriQIsqL8wYlQTJUmA8VlopECLF6L9XEDiGxvqR5zn2/u/c+WK2eWnq2eXbK3Qc+A8gOaHSc18tA/e/ff/z3iYQV/XvbgTueSghfpL/TDEI+f8XHJH/GgG1CPl/0rT+e7G+Y+//jnyT/6gP//+kRkLBUz89Bu5/9ZBb6o6MNh9BkDkQCASGwjuDucj/ABIAAA==`
)

func TestExtractZip(t *testing.T) {
	t.Run(`file nil`, func(t *testing.T) {
		fs := afero.NewMemMapFs()
		err := extractZip(fs, nil, "")
		require.EqualError(t, err, "file is nil")
	})
	t.Run(`unzip test zip file`, func(t *testing.T) {
		fs := afero.NewMemMapFs()
		zipFile := setupTestArchive(t, fs, testZip)
		defer func() { _ = zipFile.Close() }()

		err := extractZip(fs, zipFile, testDir)
		require.NoError(t, err)

		exists, err := afero.Exists(fs, filepath.Join(testDir, testFilename))
		require.NoError(t, err)
		assert.True(t, exists)

		exists, err = afero.Exists(fs, filepath.Join(testDir, testDir, testFilename))
		require.NoError(t, err)
		assert.True(t, exists)

		exists, err = afero.Exists(fs, filepath.Join(testDir, testDir, testDir, testFilename))
		require.NoError(t, err)
		assert.True(t, exists)

		exists, err = afero.Exists(fs, filepath.Join(testDir, agentConfPath, testFilename))
		require.NoError(t, err)
		assert.True(t, exists)

		info, err := fs.Stat(filepath.Join(testDir, testFilename))
		require.NoError(t, err)
		require.NotNil(t, info)
		assert.False(t, info.IsDir())
		assert.Equal(t, int64(25), info.Size())

		info, err = fs.Stat(filepath.Join(testDir, testDir, testFilename))
		require.NoError(t, err)
		require.NotNil(t, info)
		assert.False(t, info.IsDir())
		assert.Equal(t, int64(25), info.Size())

		info, err = fs.Stat(filepath.Join(testDir, testDir, testDir, testFilename))
		require.NoError(t, err)
		require.NotNil(t, info)
		assert.False(t, info.IsDir())
		assert.Equal(t, int64(25), info.Size())

		info, err = fs.Stat(filepath.Join(testDir, agentConfPath, testFilename))
		require.NoError(t, err)
		require.NotNil(t, info)
		assert.False(t, info.IsDir())
		assert.Equal(t, int64(25), info.Size())

		mode := info.Mode().Perm() & 020
		// Assert file is group writeable
		assert.NotEqual(t, mode, os.FileMode(0))
	})
}

func TestExtractGzip(t *testing.T) {
	t.Run(`path empty`, func(t *testing.T) {
		fs := afero.NewMemMapFs()
		err := extractGzip(fs, "", "")
		require.Error(t, err)
	})
	t.Run(`unzip test gzip file`, func(t *testing.T) {
		fs := afero.NewMemMapFs()
		gzipFile := setupTestArchive(t, fs, testGzip)
		defer func() { _ = gzipFile.Close() }()

		err := extractGzip(fs, gzipFile.Name(), testDir)
		require.NoError(t, err)
		// TODO: ADD MORE
	})
}

func setupTestArchive(t *testing.T, fs afero.Fs, rawZip string) afero.File {
	zipf, err := base64.StdEncoding.DecodeString(rawZip)
	require.NoError(t, err)

	zipFile, err := afero.TempFile(fs, "", "")
	require.NoError(t, err)

	_, err = zipFile.Write(zipf)
	require.NoError(t, err)

	err = zipFile.Sync()
	require.NoError(t, err)

	_, err = zipFile.Seek(0, io.SeekStart)
	require.NoError(t, err)

	return zipFile
}

func setupInavlidTestZip(t *testing.T, fs afero.Fs) afero.File {
	zipFile := setupTestArchive(t, fs, testZip)

	_, err := zipFile.Seek(8, io.SeekStart)
	require.NoError(t, err)

	return zipFile
}

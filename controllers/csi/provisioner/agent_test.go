package csiprovisioner

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	testZip      = `UEsDBAoAAAAAAKh0p1JsLSFnGQAAABkAAAAIABwAdGVzdC50eHRVVAkAA3w0lWATB55gdXgLAAEE6AMAAAToAwAAeW91IGZvdW5kIHRoZSBlYXN0ZXIgZWdnClBLAwQKAAAAAADAOa5SAAAAAAAAAAAAAAAABQAcAHRlc3QvVVQJAAMXB55gHQeeYHV4CwABBOgDAAAE6AMAAFBLAwQKAAAAAACodKdSbC0hZxkAAAAZAAAADQAcAHRlc3QvdGVzdC50eHRVVAkAA3w0lWATB55gdXgLAAEE6AMAAAToAwAAeW91IGZvdW5kIHRoZSBlYXN0ZXIgZWdnClBLAwQKAAAAAADCOa5SAAAAAAAAAAAAAAAACgAcAHRlc3QvdGVzdC9VVAkAAxwHnmAgB55gdXgLAAEE6AMAAAToAwAAUEsDBAoAAAAAAKh0p1JsLSFnGQAAABkAAAASABwAdGVzdC90ZXN0L3Rlc3QudHh0VVQJAAN8NJVgHAeeYHV4CwABBOgDAAAE6AMAAHlvdSBmb3VuZCB0aGUgZWFzdGVyIGVnZwpQSwMECgAAAAAA2zquUgAAAAAAAAAAAAAAAAYAHABhZ2VudC9VVAkAAy4JnmAxCZ5gdXgLAAEE6AMAAAToAwAAUEsDBAoAAAAAAOI6rlIAAAAAAAAAAAAAAAALABwAYWdlbnQvY29uZi9VVAkAAzgJnmA+CZ5gdXgLAAEE6AMAAAToAwAAUEsDBAoAAAAAAKh0p1JsLSFnGQAAABkAAAATABwAYWdlbnQvY29uZi90ZXN0LnR4dFVUCQADfDSVYDgJnmB1eAsAAQToAwAABOgDAAB5b3UgZm91bmQgdGhlIGVhc3RlciBlZ2cKUEsBAh4DCgAAAAAAqHSnUmwtIWcZAAAAGQAAAAgAGAAAAAAAAQAAAKSBAAAAAHRlc3QudHh0VVQFAAN8NJVgdXgLAAEE6AMAAAToAwAAUEsBAh4DCgAAAAAAwDmuUgAAAAAAAAAAAAAAAAUAGAAAAAAAAAAQAO1BWwAAAHRlc3QvVVQFAAMXB55gdXgLAAEE6AMAAAToAwAAUEsBAh4DCgAAAAAAqHSnUmwtIWcZAAAAGQAAAA0AGAAAAAAAAQAAAKSBmgAAAHRlc3QvdGVzdC50eHRVVAUAA3w0lWB1eAsAAQToAwAABOgDAABQSwECHgMKAAAAAADCOa5SAAAAAAAAAAAAAAAACgAYAAAAAAAAABAA7UH6AAAAdGVzdC90ZXN0L1VUBQADHAeeYHV4CwABBOgDAAAE6AMAAFBLAQIeAwoAAAAAAKh0p1JsLSFnGQAAABkAAAASABgAAAAAAAEAAACkgT4BAAB0ZXN0L3Rlc3QvdGVzdC50eHRVVAUAA3w0lWB1eAsAAQToAwAABOgDAABQSwECHgMKAAAAAADbOq5SAAAAAAAAAAAAAAAABgAYAAAAAAAAABAA7UGjAQAAYWdlbnQvVVQFAAMuCZ5gdXgLAAEE6AMAAAToAwAAUEsBAh4DCgAAAAAA4jquUgAAAAAAAAAAAAAAAAsAGAAAAAAAAAAQAO1B4wEAAGFnZW50L2NvbmYvVVQFAAM4CZ5gdXgLAAEE6AMAAAToAwAAUEsBAh4DCgAAAAAAqHSnUmwtIWcZAAAAGQAAABMAGAAAAAAAAQAAAKSBKAIAAGFnZW50L2NvbmYvdGVzdC50eHRVVAUAA3w0lWB1eAsAAQToAwAABOgDAABQSwUGAAAAAAgACACKAgAAjgIAAAAA`
	testDir      = "test"
	testFilename = "test.txt"
)

type failFs struct {
	afero.Fs
}

func (fs failFs) OpenFile(string, int, os.FileMode) (afero.File, error) {
	return nil, fmt.Errorf(errorMsg)
}

func TestOneAgentProvisioner_InstallAgent(t *testing.T) {
	log := logger.NewDTLogger()
	t.Run(`error when creating temp file`, func(t *testing.T) {
		fs := failFs{
			Fs: afero.NewMemMapFs(),
		}
		installAgentCfg := &installAgentConfig{
			fs:     fs,
			tenant: &metadata.Tenant{LatestVersion: ""},
		}

		err := installAgentCfg.installAgent()
		assert.EqualError(t, err, "failed to create temporary file for download: "+errorMsg)
	})
	t.Run(`error when downloading latest agent`, func(t *testing.T) {
		fs := afero.NewMemMapFs()
		dtc := &dtclient.MockDynatraceClient{}
		dtc.
			On("GetAgent", dtclient.OsUnix, dtclient.InstallerTypePaaS, dtclient.FlavorMultidistro,
				mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("*mem.File")).
			Return(fmt.Errorf(errorMsg))
		dtc.
			On("GetAgentVersions", dtclient.OsUnix, dtclient.InstallerTypePaaS, dtclient.FlavorMultidistro, mock.AnythingOfType("string")).
			Return([]string{}, fmt.Errorf(errorMsg))
		installAgentCfg := &installAgentConfig{
			fs:     fs,
			dtc:    dtc,
			logger: log,
			tenant: &metadata.Tenant{LatestVersion: ""},
		}

		err := installAgentCfg.installAgent()
		assert.EqualError(t, err, "failed to fetch latest OneAgent version: "+errorMsg)
	})
	t.Run(`error unzipping file`, func(t *testing.T) {
		fs := afero.NewMemMapFs()

		dtc := &dtclient.MockDynatraceClient{}
		dtc.
			On("GetAgent", dtclient.OsUnix, dtclient.InstallerTypePaaS, dtclient.FlavorMultidistro,
				mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("*mem.File")).
			Run(func(args mock.Arguments) {
				writer := args.Get(5).(io.Writer)

				zipFile := setupInavlidTestZip(t, fs)
				defer func() { _ = zipFile.Close() }()

				_, err := io.Copy(writer, zipFile)
				require.NoError(t, err)
			}).
			Return(nil)
		installAgentCfg := &installAgentConfig{
			fs:     fs,
			dtc:    dtc,
			logger: log,
			tenant: &metadata.Tenant{LatestVersion: ""},
		}

		err := installAgentCfg.installAgent()
		assert.Error(t, err)
	})
	t.Run(`downloading and unzipping agent`, func(t *testing.T) {
		fs := afero.NewMemMapFs()
		pathResolver := metadata.PathResolver{
			RootDir: testDir,
		}
		dtc := &dtclient.MockDynatraceClient{}
		dtc.
			On("GetAgent", dtclient.OsUnix, dtclient.InstallerTypePaaS, dtclient.FlavorMultidistro,
				mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("*mem.File")).
			Run(func(args mock.Arguments) {
				writer := args.Get(5).(io.Writer)

				zipFile := setupTestZip(t, fs)
				defer func() { _ = zipFile.Close() }()

				_, err := io.Copy(writer, zipFile)
				require.NoError(t, err)
			}).
			Return(nil)
		installAgentCfg := &installAgentConfig{
			fs:     fs,
			dtc:    dtc,
			logger: log,
			path:   pathResolver,
			tenant: &metadata.Tenant{LatestVersion: ""},
		}

		err := installAgentCfg.installAgent()
		require.NoError(t, err)

		info, err := fs.Stat(filepath.Join(pathResolver.AgentBinaryDir(""), testFilename))
		require.NoError(t, err)
		assert.NotNil(t, info)
		assert.False(t, info.IsDir())
		assert.Equal(t, int64(25), info.Size())
	})
}

func TestOneAgentProvisioner_Unzip(t *testing.T) {
	t.Run(`file nil`, func(t *testing.T) {
		fs := afero.NewMemMapFs()
		installAgentCfg := &installAgentConfig{
			path:   metadata.PathResolver{RootDir: testDir},
			fs:     fs,
			tenant: &metadata.Tenant{},
		}
		err := installAgentCfg.unzip(nil)
		require.EqualError(t, err, "file is nil")
	})
	t.Run(`unzip test zip file`, func(t *testing.T) {
		fs := afero.NewMemMapFs()
		pathResolver := metadata.PathResolver{RootDir: testDir}
		installAgentCfg := &installAgentConfig{
			path:   pathResolver,
			logger: log,
			fs:     fs,
			tenant: &metadata.Tenant{},
		}
		zipFile := setupTestZip(t, fs)
		defer func() { _ = zipFile.Close() }()

		err := installAgentCfg.unzip(zipFile)
		require.NoError(t, err)

		binaryDir := pathResolver.AgentBinaryDir("")
		exists, err := afero.Exists(fs, filepath.Join(binaryDir, testFilename))
		require.NoError(t, err)
		assert.True(t, exists)

		exists, err = afero.Exists(fs, filepath.Join(binaryDir, testDir, testFilename))
		require.NoError(t, err)
		assert.True(t, exists)

		exists, err = afero.Exists(fs, filepath.Join(binaryDir, testDir, testDir, testFilename))
		require.NoError(t, err)
		assert.True(t, exists)

		exists, err = afero.Exists(fs, filepath.Join(binaryDir, agentConfPath, testFilename))
		require.NoError(t, err)
		assert.True(t, exists)

		info, err := fs.Stat(filepath.Join(binaryDir, testFilename))
		require.NoError(t, err)
		require.NotNil(t, info)
		assert.False(t, info.IsDir())
		assert.Equal(t, int64(25), info.Size())

		info, err = fs.Stat(filepath.Join(binaryDir, testDir, testFilename))
		require.NoError(t, err)
		require.NotNil(t, info)
		assert.False(t, info.IsDir())
		assert.Equal(t, int64(25), info.Size())

		info, err = fs.Stat(filepath.Join(binaryDir, testDir, testDir, testFilename))
		require.NoError(t, err)
		require.NotNil(t, info)
		assert.False(t, info.IsDir())
		assert.Equal(t, int64(25), info.Size())

		info, err = fs.Stat(filepath.Join(binaryDir, agentConfPath, testFilename))
		require.NoError(t, err)
		require.NotNil(t, info)
		assert.False(t, info.IsDir())
		assert.Equal(t, int64(25), info.Size())

		mode := info.Mode().Perm() & 020
		// Assert file is group writeable
		assert.NotEqual(t, mode, os.FileMode(0))
	})
}

func setupTestZip(t *testing.T, fs afero.Fs) afero.File {
	zipf, err := base64.StdEncoding.DecodeString(testZip)
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
	zipFile := setupTestZip(t, fs)

	_, err := zipFile.Seek(8, io.SeekStart)
	require.NoError(t, err)

	return zipFile
}

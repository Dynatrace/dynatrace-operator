package csiprovisioner

import (
	"encoding/base64"
	"fmt"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	testError    = "test-error"
	testZip      = `UEsDBBQACAAIAKh0p1IAAAAAAAAAABkAAAAIACAAdGVzdC50eHRVVA0AB3w0lWB8NJVgfDSVYHV4CwABBOgDAAAE6AMAAKvML1VIyy/NS1EoyUhVSE0sLkktUkhNT+cCAFBLBwhsLSFnGwAAABkAAABQSwECFAMUAAgACACodKdSbC0hZxsAAAAZAAAACAAgAAAAAAAAAAAApIEAAAAAdGVzdC50eHRVVA0AB3w0lWB8NJVgfDSVYHV4CwABBOgDAAAE6AMAAFBLBQYAAAAAAQABAFYAAABxAAAAAAA=`
	testDir      = "test"
	testFilename = "test.txt"
)

type failFs struct {
	afero.Fs
}

func (fs failFs) OpenFile(string, int, os.FileMode) (afero.File, error) {
	return nil, fmt.Errorf(testError)
}

func TestOneAgentProvisioner_InstallAgent(t *testing.T) {
	log := logger.NewDTLogger()
	t.Run(`error when creating temp file`, func(t *testing.T) {
		fs := failFs{
			Fs: afero.NewMemMapFs(),
		}
		installAgentCfg := &installAgentConfig{
			fs: fs,
		}

		err := installAgent(installAgentCfg)
		assert.EqualError(t, err, "failed to create temporary file for download: "+testError)
	})
	t.Run(`error when downloading latest agent`, func(t *testing.T) {
		fs := afero.NewMemMapFs()
		dtc := &dtclient.MockDynatraceClient{}
		dtc.
			On("GetLatestAgent",
				dtclient.OsUnix, dtclient.InstallerTypePaaS,
				mock.AnythingOfType("string"), mock.AnythingOfType("string")).
			Return(ioutil.NopCloser(strings.NewReader("")), fmt.Errorf(testError))
		installAgentCfg := &installAgentConfig{
			fs:     fs,
			dtc:    dtc,
			logger: log,
		}

		err := installAgent(installAgentCfg)
		assert.EqualError(t, err, "failed to fetch latest OneAgent version: "+testError)
	})
	t.Run(`error unzipping file`, func(t *testing.T) {
		fs := afero.NewMemMapFs()
		zipFile := setupTestZip(t, fs)
		defer func() { _ = zipFile.Close() }()

		_, err := zipFile.Seek(0, io.SeekStart)
		require.NoError(t, err)

		dtc := &dtclient.MockDynatraceClient{}
		dtc.
			On("GetLatestAgent",
				dtclient.OsUnix, dtclient.InstallerTypePaaS,
				mock.AnythingOfType("string"), mock.AnythingOfType("string")).
			Return(zipFile, nil)
		installAgentCfg := &installAgentConfig{
			fs:     fs,
			dtc:    dtc,
			logger: log,
		}

		err = installAgent(installAgentCfg)
		assert.EqualError(t, err, "failed to unzip file: illegal file path: test.txt")
	})
	t.Run(`downloading and unzipping agent`, func(t *testing.T) {
		fs := afero.NewMemMapFs()
		zipFile := setupTestZip(t, fs)
		defer func() { _ = zipFile.Close() }()

		_, err := zipFile.Seek(0, io.SeekStart)
		require.NoError(t, err)

		dtc := &dtclient.MockDynatraceClient{}
		dtc.
			On("GetLatestAgent",
				dtclient.OsUnix, dtclient.InstallerTypePaaS,
				mock.AnythingOfType("string"), mock.AnythingOfType("string")).
			Return(zipFile, nil)
		installAgentCfg := &installAgentConfig{
			fs:        fs,
			dtc:       dtc,
			logger:    log,
			targetDir: testDir,
		}

		err = installAgent(installAgentCfg)
		assert.NoError(t, err)

		for _, dir := range []string{
			filepath.Join(testDir, "log"),
			filepath.Join(testDir, "datastorage"),
		} {
			info, err := fs.Stat(dir)
			assert.NoError(t, err)
			assert.NotNil(t, info)
			assert.True(t, info.IsDir())
		}

		info, err := fs.Stat(filepath.Join(testDir, testFilename))
		assert.NoError(t, err)
		assert.NotNil(t, info)
		assert.False(t, info.IsDir())
		assert.Equal(t, int64(25), info.Size())
	})
}

func setupTestZip(t *testing.T, fs afero.Fs) afero.File {
	zip, err := base64.StdEncoding.DecodeString(testZip)
	require.NoError(t, err)

	zipFile, err := afero.TempFile(fs, "", "")
	require.NoError(t, err)

	_, err = zipFile.Write(zip)
	require.NoError(t, err)

	err = zipFile.Sync()
	require.NoError(t, err)

	return zipFile
}

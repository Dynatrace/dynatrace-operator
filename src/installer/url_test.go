package installer

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	testVersion = "test"
	testUrl     = "test.url"
)

type failFs struct {
	afero.Fs
}

func (fs failFs) OpenFile(string, int, os.FileMode) (afero.File, error) {
	return nil, fmt.Errorf(testErrorMessage)
}

func TestInstallAgentFromTenant(t *testing.T) {
	t.Run(`error when creating temp file`, func(t *testing.T) {
		fs := failFs{
			Fs: afero.NewMemMapFs(),
		}
		installer := &OneAgentInstaller{
			fs: fs,
		}

		err := installer.installAgentFromUrl("")
		assert.EqualError(t, err, "failed to create temporary file for download: "+testErrorMessage)
	})
	t.Run(`error when downloading latest agent`, func(t *testing.T) {
		fs := afero.NewMemMapFs()
		dtc := &dtclient.MockDynatraceClient{}
		dtc.
			On("GetAgent", dtclient.OsUnix, dtclient.InstallerTypePaaS, dtclient.FlavorMultidistro,
				mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("[]string"), mock.AnythingOfType("*mem.File")).
			Return(fmt.Errorf(testErrorMessage))
		dtc.
			On("GetAgentVersions", dtclient.OsUnix, dtclient.InstallerTypePaaS, dtclient.FlavorMultidistro, mock.AnythingOfType("string")).
			Return([]string{}, fmt.Errorf(testErrorMessage))
		installer := &OneAgentInstaller{
			fs:  fs,
			dtc: dtc,
			props: InstallerProperties{
				Os:     dtclient.OsUnix,
				Type:   dtclient.InstallerTypePaaS,
				Flavor: dtclient.FlavorMultidistro,
			},
		}

		err := installer.installAgentFromUrl("")
		assert.EqualError(t, err, "failed to fetch OneAgent version: "+testErrorMessage)
	})
	t.Run(`error unzipping file`, func(t *testing.T) {
		fs := afero.NewMemMapFs()

		dtc := &dtclient.MockDynatraceClient{}
		dtc.
			On("GetAgent", dtclient.OsUnix, dtclient.InstallerTypePaaS, dtclient.FlavorMultidistro,
				mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("[]string"), mock.AnythingOfType("*mem.File")).
			Run(func(args mock.Arguments) {
				writer := args.Get(6).(io.Writer)

				zipFile := setupInavlidTestZip(t, fs)
				defer func() { _ = zipFile.Close() }()

				_, err := io.Copy(writer, zipFile)
				require.NoError(t, err)
			}).
			Return(nil)
		installer := &OneAgentInstaller{
			fs:  fs,
			dtc: dtc,
			props: InstallerProperties{
				Os:     dtclient.OsUnix,
				Type:   dtclient.InstallerTypePaaS,
				Flavor: dtclient.FlavorMultidistro,
			},
		}

		err := installer.installAgentFromUrl("")
		assert.Error(t, err)
	})
	t.Run(`downloading and unzipping agent via version`, func(t *testing.T) {
		fs := afero.NewMemMapFs()
		dtc := &dtclient.MockDynatraceClient{}
		dtc.
			On("GetAgent", dtclient.OsUnix, dtclient.InstallerTypePaaS, dtclient.FlavorMultidistro,
				mock.AnythingOfType("string"), testVersion, mock.AnythingOfType("[]string"), mock.AnythingOfType("*mem.File")).
			Run(func(args mock.Arguments) {
				writer := args.Get(6).(io.Writer)

				zipFile := setupTestZip(t, fs)
				defer func() { _ = zipFile.Close() }()

				_, err := io.Copy(writer, zipFile)
				require.NoError(t, err)
			}).
			Return(nil)
		installer := &OneAgentInstaller{
			fs:  fs,
			dtc: dtc,
			props: InstallerProperties{
				Os:      dtclient.OsUnix,
				Type:    dtclient.InstallerTypePaaS,
				Flavor:  dtclient.FlavorMultidistro,
				Version: testVersion,
			},
		}

		err := installer.installAgentFromUrl(testDir)
		require.NoError(t, err)

		info, err := fs.Stat(filepath.Join(testDir, testFilename))
		require.NoError(t, err)
		assert.NotNil(t, info)
		assert.False(t, info.IsDir())
		assert.Equal(t, int64(25), info.Size())
	})
	t.Run(`downloading and unzipping latest agent`, func(t *testing.T) {
		fs := afero.NewMemMapFs()
		dtc := &dtclient.MockDynatraceClient{}
		dtc.
			On("GetLatestAgent", dtclient.OsUnix, dtclient.InstallerTypePaaS, dtclient.FlavorMultidistro,
				mock.AnythingOfType("string"), mock.AnythingOfType("[]string"), mock.AnythingOfType("*mem.File")).
			Run(func(args mock.Arguments) {
				writer := args.Get(5).(io.Writer)

				zipFile := setupTestZip(t, fs)
				defer func() { _ = zipFile.Close() }()

				_, err := io.Copy(writer, zipFile)
				require.NoError(t, err)
			}).
			Return(nil)
		installer := &OneAgentInstaller{
			fs:  fs,
			dtc: dtc,
			props: InstallerProperties{
				Os:      dtclient.OsUnix,
				Type:    dtclient.InstallerTypePaaS,
				Flavor:  dtclient.FlavorMultidistro,
				Version: VersionLatest,
			},
		}

		err := installer.installAgentFromUrl(testDir)
		require.NoError(t, err)

		info, err := fs.Stat(filepath.Join(testDir, testFilename))
		require.NoError(t, err)
		assert.NotNil(t, info)
		assert.False(t, info.IsDir())
		assert.Equal(t, int64(25), info.Size())
	})
	t.Run(`downloading and unzipping agent via url`, func(t *testing.T) {
		fs := afero.NewMemMapFs()
		dtc := &dtclient.MockDynatraceClient{}
		dtc.
			On("GetAgentViaInstallerUrl", testUrl, mock.AnythingOfType("*mem.File")).
			Run(func(args mock.Arguments) {
				writer := args.Get(1).(io.Writer)

				zipFile := setupTestZip(t, fs)
				defer func() { _ = zipFile.Close() }()

				_, err := io.Copy(writer, zipFile)
				require.NoError(t, err)
			}).
			Return(nil)
		installer := &OneAgentInstaller{
			fs:  fs,
			dtc: dtc,
			props: InstallerProperties{
				Url: testUrl,
			},
		}

		err := installer.installAgentFromUrl(testDir)
		require.NoError(t, err)

		info, err := fs.Stat(filepath.Join(testDir, testFilename))
		require.NoError(t, err)
		assert.NotNil(t, info)
		assert.False(t, info.IsDir())
		assert.Equal(t, int64(25), info.Size())
	})
}

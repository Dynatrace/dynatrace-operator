package url

import (
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/arch"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/zip"
	"github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	testVersion = "test"
	testUrl     = "test.url"

	testDir          = "test"
	testErrorMessage = "BOOM"
)

type failFs struct {
	afero.Fs
}

func (fs failFs) OpenFile(string, int, os.FileMode) (afero.File, error) {
	return nil, fmt.Errorf(testErrorMessage)
}

func TestInstallAgentFromUrl(t *testing.T) {
	t.Run(`error when creating temp file`, func(t *testing.T) {
		fs := failFs{
			Fs: afero.NewMemMapFs(),
		}
		installer := &Installer{
			fs: fs,
		}

		err := installer.installAgent("")
		assert.EqualError(t, err, testErrorMessage)
	})
	t.Run(`error when downloading latest agent`, func(t *testing.T) {
		fs := afero.NewMemMapFs()
		dtc := mocks.NewClient(t)
		dtc.
			On("GetAgent", dtclient.OsUnix, dtclient.InstallerTypePaaS, arch.FlavorMultidistro,
				mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("[]string"),
				mock.AnythingOfType("bool"), mock.AnythingOfType("*mem.File")).
			Return(fmt.Errorf(testErrorMessage))
		dtc.
			On("GetAgentVersions", dtclient.OsUnix, dtclient.InstallerTypePaaS, arch.FlavorMultidistro, mock.AnythingOfType("string")).
			Return([]string{}, fmt.Errorf(testErrorMessage))
		installer := &Installer{
			fs:  fs,
			dtc: dtc,
			props: &Properties{
				Os:     dtclient.OsUnix,
				Type:   dtclient.InstallerTypePaaS,
				Flavor: arch.FlavorMultidistro,
			},
		}

		err := installer.installAgent("")
		assert.EqualError(t, err, testErrorMessage)
	})
	t.Run(`error unzipping file`, func(t *testing.T) {
		fs := afero.NewMemMapFs()

		dtc := mocks.NewClient(t)
		dtc.
			On("GetAgent", dtclient.OsUnix, dtclient.InstallerTypePaaS, arch.FlavorMultidistro,
				mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("[]string"),
				mock.AnythingOfType("bool"), mock.AnythingOfType("*mem.File")).
			Run(func(args mock.Arguments) {
				writer, _ := args.Get(7).(io.Writer)

				zipFile := zip.SetupInvalidTestZip(t, fs)
				defer func() { _ = zipFile.Close() }()

				_, err := io.Copy(writer, zipFile)
				require.NoError(t, err)
			}).
			Return(nil)
		installer := &Installer{
			fs:        fs,
			dtc:       dtc,
			extractor: zip.NewOneAgentExtractor(fs, metadata.PathResolver{}),
			props: &Properties{
				Os:     dtclient.OsUnix,
				Type:   dtclient.InstallerTypePaaS,
				Flavor: arch.FlavorMultidistro,
			},
		}

		err := installer.installAgent("")
		assert.Error(t, err)
	})
	t.Run(`downloading and unzipping agent via version`, func(t *testing.T) {
		fs := afero.NewMemMapFs()
		dtc := mocks.NewClient(t)
		dtc.
			On("GetAgent", dtclient.OsUnix, dtclient.InstallerTypePaaS, arch.FlavorMultidistro,
				mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("[]string"),
				mock.AnythingOfType("bool"), mock.AnythingOfType("*mem.File")).
			Run(func(args mock.Arguments) {
				writer, _ := args.Get(7).(io.Writer)

				zipFile := zip.SetupTestArchive(t, fs, zip.TestRawZip)
				defer func() { _ = zipFile.Close() }()

				_, err := io.Copy(writer, zipFile)
				require.NoError(t, err)
			}).
			Return(nil)
		installer := &Installer{
			fs:        fs,
			dtc:       dtc,
			extractor: zip.NewOneAgentExtractor(fs, metadata.PathResolver{}),
			props: &Properties{
				Os:            dtclient.OsUnix,
				Type:          dtclient.InstallerTypePaaS,
				Flavor:        arch.FlavorMultidistro,
				TargetVersion: testVersion,
			},
		}

		err := installer.installAgent(testDir)
		require.NoError(t, err)
		// afero can't rename directories properly: https://github.com/spf13/afero/issues/141
	})
	t.Run(`downloading and unzipping latest agent`, func(t *testing.T) {
		fs := afero.NewMemMapFs()
		dtc := mocks.NewClient(t)
		dtc.
			On("GetLatestAgent", dtclient.OsUnix, dtclient.InstallerTypePaaS, arch.FlavorMultidistro,
				mock.AnythingOfType("string"), mock.AnythingOfType("[]string"), mock.AnythingOfType("bool"),
				mock.AnythingOfType("*mem.File")).
			Run(func(args mock.Arguments) {
				writer, _ := args.Get(6).(io.Writer)

				zipFile := zip.SetupTestArchive(t, fs, zip.TestRawZip)
				defer func() { _ = zipFile.Close() }()

				_, err := io.Copy(writer, zipFile)
				require.NoError(t, err)
			}).
			Return(nil)
		installer := &Installer{
			fs:        fs,
			dtc:       dtc,
			extractor: zip.NewOneAgentExtractor(fs, metadata.PathResolver{}),
			props: &Properties{
				Os:            dtclient.OsUnix,
				Type:          dtclient.InstallerTypePaaS,
				Flavor:        arch.FlavorMultidistro,
				TargetVersion: VersionLatest,
			},
		}

		err := installer.installAgent(testDir)
		require.NoError(t, err)
		// afero can't rename directories properly: https://github.com/spf13/afero/issues/141
	})
	t.Run(`downloading and unzipping agent via url`, func(t *testing.T) {
		fs := afero.NewMemMapFs()
		dtc := mocks.NewClient(t)
		dtc.
			On("GetAgentViaInstallerUrl", testUrl, mock.AnythingOfType("*mem.File")).
			Run(func(args mock.Arguments) {
				writer, _ := args.Get(1).(io.Writer)

				zipFile := zip.SetupTestArchive(t, fs, zip.TestRawZip)
				defer func() { _ = zipFile.Close() }()

				_, err := io.Copy(writer, zipFile)
				require.NoError(t, err)
			}).
			Return(nil)
		installer := &Installer{
			fs:        fs,
			dtc:       dtc,
			extractor: zip.NewOneAgentExtractor(fs, metadata.PathResolver{}),
			props: &Properties{
				Url: testUrl,
			},
		}

		err := installer.installAgent(testDir)
		require.NoError(t, err)
		// afero can't rename directories properly: https://github.com/spf13/afero/issues/141
	})
}

func TestIsAlreadyDownloaded(t *testing.T) {
	t.Run(`true if exits`, func(t *testing.T) {
		fs := afero.NewMemMapFs()
		targetDir := "test/test"
		fs.MkdirAll(targetDir, 0666)
		installer := &Installer{
			fs: fs,
		}
		assert.True(t, installer.isAlreadyDownloaded(targetDir))
	})
	t.Run(`false if standalone`, func(t *testing.T) {
		fs := afero.NewMemMapFs()
		targetDir := consts.AgentBinDirMount
		installer := &Installer{
			fs:    fs,
			props: &Properties{},
		}
		assert.False(t, installer.isAlreadyDownloaded(targetDir))
	})
}

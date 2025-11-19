package url

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/arch"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/zip"
	dtclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	testVersion = "test"
	testURL     = "test.url"

	testErrorMessage = "BOOM"
)

func TestInstallAgentFromUrl(t *testing.T) {
	ctx := context.Background()

	t.Run("error when creating temp file", func(t *testing.T) {
		problematicFolder := filepath.Join(t.TempDir(), "boom")
		require.NoError(t, os.MkdirAll(problematicFolder, 0444)) // r--r--r--, "readonly"

		t.Cleanup(func() {
			// needed, otherwise the `problematicFolder` wont be cleaned up after the test
			os.Chmod(problematicFolder, 0755)
		})
		installer := &Installer{}

		err := installer.installAgent(ctx, filepath.Join(problematicFolder, "target"))
		require.Error(t, err)
	})
	t.Run("error when downloading latest agent", func(t *testing.T) {
		target := filepath.Join(t.TempDir(), "target")
		dtc := dtclientmock.NewClient(t)
		dtc.
			On("GetAgent", mock.AnythingOfType("context.backgroundCtx"), dtclient.OsUnix, dtclient.InstallerTypePaaS, arch.FlavorMultidistro,
				mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("[]string"),
				mock.AnythingOfType("bool"), mock.AnythingOfType("*os.File")).
			Return(errors.New(testErrorMessage))
		dtc.
			On("GetAgentVersions", mock.AnythingOfType("context.backgroundCtx"), dtclient.OsUnix, dtclient.InstallerTypePaaS, arch.FlavorMultidistro, mock.AnythingOfType("string")).
			Return([]string{}, errors.New(testErrorMessage))

		installer := &Installer{
			dtc: dtc,
			props: &Properties{
				Os:     dtclient.OsUnix,
				Type:   dtclient.InstallerTypePaaS,
				Flavor: arch.FlavorMultidistro,
			},
		}

		err := installer.installAgent(ctx, target)
		require.EqualError(t, err, testErrorMessage)
	})
	t.Run("error unzipping file", func(t *testing.T) {
		tmpDir := t.TempDir()
		target := filepath.Join(tmpDir, "target")
		dtc := dtclientmock.NewClient(t)
		dtc.
			On("GetAgent", mock.AnythingOfType("context.backgroundCtx"), dtclient.OsUnix, dtclient.InstallerTypePaaS, arch.FlavorMultidistro,
				mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("[]string"),
				mock.AnythingOfType("bool"), mock.AnythingOfType("*os.File")).
			Run(func(args mock.Arguments) {
				writer, _ := args.Get(8).(io.Writer)

				zipFile := zip.SetupInvalidTestZip(t, tmpDir)
				defer func() { _ = zipFile.Close() }()

				_, err := io.Copy(writer, zipFile)
				require.NoError(t, err)
			}).
			Return(nil)

		installer := &Installer{
			dtc:       dtc,
			extractor: zip.NewOneAgentExtractor(metadata.PathResolver{RootDir: tmpDir}),
			props: &Properties{
				Os:     dtclient.OsUnix,
				Type:   dtclient.InstallerTypePaaS,
				Flavor: arch.FlavorMultidistro,
			},
		}

		err := installer.installAgent(ctx, target)
		require.Error(t, err)
	})
	t.Run("downloading and unzipping agent via version", func(t *testing.T) {
		tmpDir := t.TempDir()
		target := filepath.Join(tmpDir, testVersion)
		dtc := dtclientmock.NewClient(t)
		dtc.On("GetAgent",
			mock.AnythingOfType("context.backgroundCtx"),
			dtclient.OsUnix,
			dtclient.InstallerTypePaaS,
			arch.FlavorMultidistro,
			mock.AnythingOfType("string"),
			mock.AnythingOfType("string"),
			mock.AnythingOfType("[]string"),
			mock.AnythingOfType("bool"),
			mock.AnythingOfType("*os.File"),
		).
			Run(func(args mock.Arguments) {
				writer, _ := args.Get(8).(io.Writer)

				zipFile := zip.SetupTestArchive(t, zip.TestRawZip)
				defer func() { _ = zipFile.Close() }()

				_, err := io.Copy(writer, zipFile)
				require.NoError(t, err)
			}).
			Return(nil)

		installer := &Installer{
			dtc:       dtc,
			extractor: zip.NewOneAgentExtractor(metadata.PathResolver{RootDir: tmpDir}),
			props: &Properties{
				Os:            dtclient.OsUnix,
				Type:          dtclient.InstallerTypePaaS,
				Flavor:        arch.FlavorMultidistro,
				TargetVersion: testVersion,
			},
		}

		err := installer.installAgent(ctx, target)
		require.NoError(t, err)
	})
	t.Run("downloading and unzipping latest agent", func(t *testing.T) {
		tmpDir := t.TempDir()
		target := filepath.Join(tmpDir, VersionLatest)
		dtc := dtclientmock.NewClient(t)
		dtc.
			On("GetLatestAgent", mock.AnythingOfType("context.backgroundCtx"), dtclient.OsUnix, dtclient.InstallerTypePaaS, arch.FlavorMultidistro,
				mock.AnythingOfType("string"), mock.AnythingOfType("[]string"), mock.AnythingOfType("bool"),
				mock.AnythingOfType("*os.File")).
			Run(func(args mock.Arguments) {
				writer, _ := args.Get(7).(io.Writer)

				zipFile := zip.SetupTestArchive(t, zip.TestRawZip)
				defer func() { _ = zipFile.Close() }()

				_, err := io.Copy(writer, zipFile)
				require.NoError(t, err)
			}).
			Return(nil)

		installer := &Installer{
			dtc:       dtc,
			extractor: zip.NewOneAgentExtractor(metadata.PathResolver{RootDir: tmpDir}),
			props: &Properties{
				Os:            dtclient.OsUnix,
				Type:          dtclient.InstallerTypePaaS,
				Flavor:        arch.FlavorMultidistro,
				TargetVersion: VersionLatest,
			},
		}

		err := installer.installAgent(ctx, target)
		require.NoError(t, err)
	})
	t.Run("downloading and unzipping agent via url", func(t *testing.T) {
		tmpDir := t.TempDir()
		target := filepath.Join(tmpDir, VersionLatest)
		dtc := dtclientmock.NewClient(t)
		dtc.
			On("GetAgentViaInstallerURL", mock.AnythingOfType("context.backgroundCtx"), testURL, mock.AnythingOfType("*os.File")).
			Run(func(args mock.Arguments) {
				writer, _ := args.Get(2).(io.Writer)

				zipFile := zip.SetupTestArchive(t, zip.TestRawZip)
				defer func() { _ = zipFile.Close() }()

				_, err := io.Copy(writer, zipFile)
				require.NoError(t, err)
			}).
			Return(nil)

		installer := &Installer{
			dtc:       dtc,
			extractor: zip.NewOneAgentExtractor(metadata.PathResolver{RootDir: tmpDir}),
			props: &Properties{
				URL: testURL,
			},
		}

		err := installer.installAgent(ctx, target)
		require.NoError(t, err)
	})
}

func TestIsAlreadyDownloaded(t *testing.T) {
	t.Run("true if exits", func(t *testing.T) {
		targetDir := filepath.Join(t.TempDir(), "test")
		err := os.MkdirAll(targetDir, 0666)
		require.NoError(t, err)

		installer := &Installer{}
		assert.True(t, installer.isAlreadyDownloaded(targetDir))
	})
	t.Run("false if standalone", func(t *testing.T) {
		targetDir := filepath.Join(t.TempDir(), consts.AgentInitBinDirMount)
		installer := &Installer{
			props: &Properties{},
		}
		assert.False(t, installer.isAlreadyDownloaded(targetDir))
	})
}

package url

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/arch"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/installer"
	oneagentclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/zip"
	oneagentclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/oneagent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	testVersion = "test"

	testErrorMessage = "BOOM"
)

func TestInstallAgentFromUrl(t *testing.T) {
	getParams := oneagentclient.GetParams{
		OS:            installer.OSUnix,
		InstallerType: installer.TypePaaS,
		Flavor:        arch.FlavorMultidistro,
		Version:       "",
		Technologies:  nil,
		SkipMetadata:  false,
	}

	t.Run("error when creating temp file", func(t *testing.T) {
		problematicFolder := filepath.Join(t.TempDir(), "boom")
		require.NoError(t, os.MkdirAll(problematicFolder, 0444)) // r--r--r--, "readonly"

		t.Cleanup(func() {
			// needed, otherwise the `problematicFolder` wont be cleaned up after the test
			os.Chmod(problematicFolder, 0755)
		})
		installer := &Installer{}

		err := installer.installAgent(t.Context(), filepath.Join(problematicFolder, "target"))
		require.Error(t, err)
	})
	t.Run("error when downloading latest agent", func(t *testing.T) {
		target := filepath.Join(t.TempDir(), "target")
		dtClient := oneagentclientmock.NewAPIClient(t)
		dtClient.EXPECT().Get(t.Context(), getParams, mock.AnythingOfType("*os.File")).
			Return(errors.New(testErrorMessage))

		dtClient.EXPECT().GetVersions(t.Context(), getParams).
			Return([]string{}, errors.New(testErrorMessage))

		installer := &Installer{
			dtClient: dtClient,
			props: &Properties{
				OS:     installer.OSUnix,
				Type:   installer.TypePaaS,
				Flavor: arch.FlavorMultidistro,
			},
		}

		err := installer.installAgent(t.Context(), target)
		require.EqualError(t, err, testErrorMessage)
	})
	t.Run("error unzipping file", func(t *testing.T) {
		tmpDir := t.TempDir()
		target := filepath.Join(tmpDir, "target")
		dtClient := oneagentclientmock.NewAPIClient(t)
		dtClient.EXPECT().Get(t.Context(), getParams, mock.AnythingOfType("*os.File")).
			Run(func(ctx context.Context, args oneagentclient.GetParams, writer io.Writer) {
				zipFile := zip.SetupInvalidTestZip(t, tmpDir)
				defer func() { _ = zipFile.Close() }()

				_, err := io.Copy(writer, zipFile)
				require.NoError(t, err)
			}).
			Return(nil)

		installer := &Installer{
			dtClient:  dtClient,
			extractor: zip.NewOneAgentExtractor(metadata.PathResolver{RootDir: tmpDir}),
			props: &Properties{
				OS:     installer.OSUnix,
				Type:   installer.TypePaaS,
				Flavor: arch.FlavorMultidistro,
			},
		}

		err := installer.installAgent(t.Context(), target)
		require.Error(t, err)
	})
	t.Run("downloading and unzipping agent via version", func(t *testing.T) {
		getParams.Version = testVersion
		t.Cleanup(func() { getParams.Version = "" })

		tmpDir := t.TempDir()
		target := filepath.Join(tmpDir, testVersion)
		dtClient := oneagentclientmock.NewAPIClient(t)
		dtClient.EXPECT().Get(
			t.Context(),
			getParams,
			mock.AnythingOfType("*os.File"),
		).
			Run(func(ctx context.Context, args oneagentclient.GetParams, writer io.Writer) {
				zipFile := zip.SetupTestArchive(t, zip.TestRawZip)
				defer func() { _ = zipFile.Close() }()

				_, err := io.Copy(writer, zipFile)
				require.NoError(t, err)
			}).
			Return(nil)

		installer := &Installer{
			dtClient:  dtClient,
			extractor: zip.NewOneAgentExtractor(metadata.PathResolver{RootDir: tmpDir}),
			props: &Properties{
				OS:            installer.OSUnix,
				Type:          installer.TypePaaS,
				Flavor:        arch.FlavorMultidistro,
				TargetVersion: testVersion,
			},
		}

		err := installer.installAgent(t.Context(), target)
		require.NoError(t, err)
	})
	t.Run("downloading and unzipping latest agent", func(t *testing.T) {
		tmpDir := t.TempDir()
		target := filepath.Join(tmpDir, VersionLatest)
		dtClient := oneagentclientmock.NewAPIClient(t)
		dtClient.EXPECT().GetLatest(t.Context(), getParams,
			mock.AnythingOfType("*os.File")).
			Run(func(ctx context.Context, args oneagentclient.GetParams, writer io.Writer) {
				zipFile := zip.SetupTestArchive(t, zip.TestRawZip)
				defer func() { _ = zipFile.Close() }()

				_, err := io.Copy(writer, zipFile)
				require.NoError(t, err)
			}).
			Return(nil)

		installer := &Installer{
			dtClient:  dtClient,
			extractor: zip.NewOneAgentExtractor(metadata.PathResolver{RootDir: tmpDir}),
			props: &Properties{
				OS:            installer.OSUnix,
				Type:          installer.TypePaaS,
				Flavor:        arch.FlavorMultidistro,
				TargetVersion: VersionLatest,
			},
		}

		err := installer.installAgent(t.Context(), target)
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

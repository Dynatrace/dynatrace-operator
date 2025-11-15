package download

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Dynatrace/dynatrace-bootstrapper/pkg/configure/oneagent/ca"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/url"
	dtclientmocks "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace"
	installermock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/injection/codemodule/installer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		client := New()

		assert.NotNil(t, client.newInstaller)
		assert.NotNil(t, client.newDTClient)
	})

	t.Run("options", func(t *testing.T) {
		props := &url.Properties{
			Os:   "os",
			Arch: "arch",
		}
		opts := []Option{
			WithDTClient(dtClientTester(t, []dtclient.Option{}...)),
			WithInstaller(installerTester(t, props, nil)),
		}
		client := New(opts...)

		require.NotNil(t, client.newInstaller)
		require.NotNil(t, client.newDTClient)

		dtClient, err := client.newDTClient("url", "api", "paas", []dtclient.Option{}...)
		require.NoError(t, err)
		require.NotNil(t, dtClient)
		require.IsType(t, &dtclientmocks.Client{}, dtClient)

		installer := client.newInstaller(dtClient, props)
		require.NotNil(t, installer)
		require.IsType(t, &installermock.Installer{}, installer)
	})
}

func TestDo(t *testing.T) {
	ctx := context.Background()

	t.Run("no config ==> error", func(t *testing.T) {
		tmpDir := t.TempDir()
		inputDir := filepath.Join(tmpDir, "input")
		targetDir := filepath.Join(tmpDir, "target")

		opts := []Option{
			WithDTClient(dtClientTester(t, []dtclient.Option{}...)),
			WithInstaller(installerTester(t, &url.Properties{}, nil)),
		}
		client := New(opts...)

		err := client.Do(ctx, inputDir, targetDir, url.Properties{})
		require.Error(t, err)
	})

	t.Run("happy path", func(t *testing.T) {
		tmpDir := t.TempDir()
		inputDir := filepath.Join(tmpDir, "input")
		targetDir := filepath.Join(tmpDir, "target")
		config := testConfig(t)
		props := &url.Properties{
			Os:   "os",
			Arch: "arch",
		}
		setupConfig(t, inputDir, config)

		opts := []Option{
			WithDTClient(dtClientTester(t, config.toDTClientOptions()...)),
			WithInstaller(installerTester(t, props, func(i *installermock.Installer) {
				i.On("InstallAgent", mock.AnythingOfType("context.backgroundCtx"), targetDir).Return(true, nil)
			})),
		}
		client := New(opts...)

		err := client.Do(ctx, inputDir, targetDir, *props)
		require.NoError(t, err)
	})

	t.Run("certs available ==> extra option", func(t *testing.T) {
		tmpDir := t.TempDir()
		inputDir := filepath.Join(tmpDir, "input")
		targetDir := filepath.Join(tmpDir, "target")
		config := testConfig(t)
		props := &url.Properties{
			Os:   "os",
			Arch: "arch",
		}
		os.MkdirAll(inputDir, os.ModePerm)
		err := os.WriteFile(filepath.Join(inputDir, ca.TrustedCertsInputFile), []byte("cert"), 0600)
		require.NoError(t, err)

		setupConfig(t, inputDir, config)

		expectedOpts := config.toDTClientOptions()
		expectedOpts = append(expectedOpts, dtclient.Certs([]byte("cert")))

		opts := []Option{
			WithDTClient(dtClientTester(t, expectedOpts...)),
			WithInstaller(installerTester(t, props, func(i *installermock.Installer) {
				i.On("InstallAgent", mock.AnythingOfType("context.backgroundCtx"), targetDir).Return(true, nil)
			})),
		}
		client := New(opts...)

		err = client.Do(ctx, inputDir, targetDir, *props)
		require.NoError(t, err)
	})

	t.Run("installer error ==> error", func(t *testing.T) {
		tmpDir := t.TempDir()
		inputDir := filepath.Join(tmpDir, "input")
		targetDir := filepath.Join(tmpDir, "target")
		config := testConfig(t)
		props := &url.Properties{
			Os:   "os",
			Arch: "arch",
		}
		setupConfig(t, inputDir, config)

		expectedErr := errors.New("boom")

		opts := []Option{
			WithDTClient(dtClientTester(t, config.toDTClientOptions()...)),
			WithInstaller(installerTester(t, props, func(i *installermock.Installer) {
				i.On("InstallAgent", mock.AnythingOfType("context.backgroundCtx"), targetDir).Return(false, expectedErr)
			})),
		}
		client := New(opts...)

		err := client.Do(ctx, inputDir, targetDir, *props)
		require.Error(t, err)
		require.ErrorIs(t, err, expectedErr)
	})
}

type mockConfigFunc func(*installermock.Installer)

func installerTester(t *testing.T, expectedProps *url.Properties, mockFunc mockConfigFunc) url.NewFunc {
	t.Helper()

	return func(dtc dtclient.Client, props *url.Properties) installer.Installer {
		require.NotNil(t, dtc)
		require.NotEmpty(t, props)
		require.Equal(t, *expectedProps, *props)

		mock := installermock.NewInstaller(t)

		if mockFunc != nil {
			mockFunc(mock)
		}

		return mock
	}
}

func dtClientTester(t *testing.T, expectedOpts ...dtclient.Option) dtclient.NewFunc {
	t.Helper()

	return func(url, apiToken, paasToken string, opts ...dtclient.Option) (dtclient.Client, error) {
		require.NotEmpty(t, url)
		require.NotEmpty(t, apiToken)
		require.NotEmpty(t, paasToken)

		compareDTOptions(t, expectedOpts, opts)

		return dtclientmocks.NewClient(t), nil
	}
}

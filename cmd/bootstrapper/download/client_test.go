package download

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-bootstrapper/pkg/configure/oneagent/ca"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/url"
	installermock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/injection/codemodule/installer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		client := New()

		assert.NotNil(t, client.newInstaller)
	})

	t.Run("options", func(t *testing.T) {
		props := &url.Properties{
			OS:   "os",
			Arch: "arch",
		}
		opts := []Option{
			WithInstaller(installerTester(t, props, nil)),
		}
		client := New(opts...)

		require.NotNil(t, client.newInstaller)

		dtClient, err := dtclient.NewClientV2(dtclient.WithBaseURL("url"))
		require.NoError(t, err)
		require.NotNil(t, dtClient)

		installer := client.newInstaller(dtClient.OneAgent, props)
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
			OS:   "os",
			Arch: "arch",
		}
		setupConfig(t, inputDir, config)

		opts := []Option{
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
			OS:   "os",
			Arch: "arch",
		}
		os.MkdirAll(inputDir, os.ModePerm)
		cert := fakeCert(t)
		err := os.WriteFile(filepath.Join(inputDir, ca.TrustedCertsInputFile), cert, 0600)
		require.NoError(t, err)

		setupConfig(t, inputDir, config)

		opts := []Option{
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
			OS:   "os",
			Arch: "arch",
		}
		setupConfig(t, inputDir, config)

		expectedErr := errors.New("boom")

		opts := []Option{
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

	return func(dtc oneagent.APIClient, props *url.Properties) installer.Installer {
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

func fakeCert(t *testing.T) []byte {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    time.Now().Add(-10 * time.Second),
		NotAfter:     time.Now().Add(10 * time.Second),
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

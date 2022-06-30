package standalone

import (
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetLDPreload(t *testing.T) {
	runner := createMockedOneAgentSetup(t)
	t.Run(`create ld preload file`, func(t *testing.T) {
		runner.fs = afero.NewMemMapFs()

		err := runner.setLDPreload()

		require.NoError(t, err)
		assertIfFileExists(t,
			runner.fs,
			filepath.Join(
				ShareDirMount,
				ldPreloadFilename))
	})
}

func TestPropagateTLSCert(t *testing.T) {
	runner := createMockedOneAgentSetup(t)
	runner.config.HasHost = false

	t.Run(`create tls custom.pem`, func(t *testing.T) {
		runner.fs = afero.NewMemMapFs()

		err := runner.propagateTLSCert()

		require.NoError(t, err)
		assertIfFileExists(t,
			runner.fs,
			filepath.Join(ShareDirMount, "custom.pem"))
	})
}

func TestWriteCurlOptions(t *testing.T) {
	filesystem := afero.NewMemMapFs()
	runner := oneAgentSetup{
		config: &SecretConfig{InitialConnectRetry: 30},
		env:    &environment{OneAgentInjected: true},
		fs:     filesystem,
	}

	err := runner.configureInstallation()

	assert.NoError(t, err)

	exists, err := afero.Exists(filesystem, "mnt/share/curl_options.conf")

	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestCurlOptions(t *testing.T) {
	filesystem := afero.NewMemMapFs()
	setup := oneAgentSetup{
		config: &SecretConfig{InitialConnectRetry: 30},
		fs:     filesystem,
	}

	assert.Equal(t, "initialConnectRetryMs 30\n", setup.getCurlOptionsContent())

	err := setup.createCurlOptionsFile()

	assert.NoError(t, err)

	exists, err := afero.Exists(filesystem, "mnt/share/curl_options.conf")

	assert.NoError(t, err)
	assert.True(t, exists)

	content, err := afero.ReadFile(filesystem, "mnt/share/curl_options.conf")

	assert.NoError(t, err)
	assert.Equal(t, setup.getCurlOptionsContent(), string(content))
}

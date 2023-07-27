package dockerkeychain

import (
	"encoding/json"
	"io/fs"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	registryName         = "docker.test.com"
	dockerConfigFilename = "/docker.config"
	testToken            = "test-token"
	testPassword         = "test-password"
	testAuth             = "dGVzdC10b2tlbjp0ZXN0LXBhc3N3b3Jk" // echo -n "test-token:test-password" | base64
	dockerConfig         = "{\"auths\":{\"" + registryName + "\":{\"username\":\"" + testToken + "\",\"password\":\"" + testPassword + "\",\"auth\":\"" + testAuth + "\"}}}"
)

func TestNewDockerKeychain(t *testing.T) {
	t.Run("config file not found", func(t *testing.T) {
		memFs := afero.NewMemMapFs()
		keychain := NewDockerKeychain(dockerConfigFilename, memFs)
		registry, err := name.NewRegistry(registryName, name.StrictValidation)
		require.NoError(t, err)

		authenticator, err := keychain.Resolve(registry)

		require.Error(t, err)
		var pathError *fs.PathError
		ok := errors.As(err, &pathError)
		require.True(t, ok)
		assert.Equal(t, "open", pathError.Op)

		assert.Nil(t, authenticator)
	})

	t.Run("invalid format", func(t *testing.T) {
		memFs := afero.NewMemMapFs()
		configFile, err := memFs.Create(dockerConfigFilename)
		require.NoError(t, err)
		_, err = configFile.WriteString("invalid format")
		require.NoError(t, err)
		err = configFile.Close()
		require.NoError(t, err)

		keychain := NewDockerKeychain(dockerConfigFilename, memFs)
		registry, err := name.NewRegistry(registryName, name.StrictValidation)
		require.NoError(t, err)

		authenticator, err := keychain.Resolve(registry)

		require.Error(t, err)
		var syntaxError *json.SyntaxError
		ok := errors.As(err, &syntaxError)
		require.True(t, ok)
		assert.Equal(t, int64(1), syntaxError.Offset)

		assert.Nil(t, authenticator)
	})

	t.Run("valid config file", func(t *testing.T) {
		memFs := afero.NewMemMapFs()
		configFile, err := memFs.Create(dockerConfigFilename)
		require.NoError(t, err)
		_, err = configFile.WriteString(dockerConfig)
		require.NoError(t, err)
		err = configFile.Close()
		require.NoError(t, err)

		keychain := NewDockerKeychain(dockerConfigFilename, memFs)
		registry, err := name.NewRegistry(registryName, name.StrictValidation)
		require.NoError(t, err)

		authenticator, err := keychain.Resolve(registry)

		require.NoError(t, err)
		assert.NotNil(t, authenticator)
		auth, err := authenticator.Authorization()
		require.NoError(t, err)
		assert.Equal(t, testToken, auth.Username)
		assert.Equal(t, testPassword, auth.Password)
	})
}

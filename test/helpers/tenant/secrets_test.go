//go:build e2e

package tenant

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecretFileContent = "apiUrl: apiUrl\napiToken: apiToken"

func TestNewFromConfig(t *testing.T) {
	fs := afero.NewMemMapFs()
	workingDir, err := os.Getwd()

	require.NoError(t, err)

	secretsPath := filepath.Join(workingDir, "..", "testdata", "Secrets")
	require.NoError(t, fs.MkdirAll(secretsPath, 0655))

	require.NoError(t, afero.WriteFile(fs, filepath.Join(secretsPath, "Secrets-test.yaml"),
		[]byte(testSecretFileContent), 0755))

	tenantSecrets, err := newFromConfig(fs, filepath.Join(secretsPath, "Secrets-test.yaml"))

	require.NoError(t, err)
	assert.Equal(t, "apiUrl", tenantSecrets.APIURL)
	assert.Equal(t, "apiToken", tenantSecrets.APIToken)
}

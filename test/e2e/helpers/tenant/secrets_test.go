//go:build e2e

package tenant

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecretFileContent = "apiUrl: apiUrl\napiToken: apiToken"

func TestNewFromConfig(t *testing.T) {
	workingDir := t.TempDir()

	secretsPath := filepath.Join(workingDir, "..", "testdata", "Secrets")
	require.NoError(t, os.MkdirAll(secretsPath, 0655))

	require.NoError(t, os.WriteFile(filepath.Join(secretsPath, "Secrets-test.yaml"),
		[]byte(testSecretFileContent), 0600))

	tenantSecrets, err := newFromConfig(filepath.Join(secretsPath, "Secrets-test.yaml"))

	require.NoError(t, err)
	assert.Equal(t, "apiUrl", tenantSecrets.APIURL)
	assert.Equal(t, "apiToken", tenantSecrets.APIToken)
}

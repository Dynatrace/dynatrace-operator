//go:build e2e

package support_archive

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/secrets"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func getSecretConfig(t *testing.T) secrets.Secret {
	secretConfig, err := secrets.DefaultSingleTenant(afero.NewOsFs())
	require.NoError(t, err)
	return secretConfig
}

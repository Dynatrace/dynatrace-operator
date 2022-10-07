package dynakube

import (
	"os"
	"path"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/secrets"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

const (
	installSecretsPath = "../testdata/secrets/install.yaml"
)

func GetSecretConfig(t *testing.T) secrets.Secret {
	currentWorkingDirectory, err := os.Getwd()
	require.NoError(t, err)

	secretPath := path.Join(currentWorkingDirectory, installSecretsPath)
	secretConfig, err := secrets.NewFromConfig(afero.NewOsFs(), secretPath)

	require.NoError(t, err)

	return secretConfig
}

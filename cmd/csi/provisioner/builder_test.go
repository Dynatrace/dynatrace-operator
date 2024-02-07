package provisioner

import (
	"io/fs"
	"testing"

	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	configmock "github.com/Dynatrace/dynatrace-operator/test/mocks/cmd/config"
	managermock "github.com/Dynatrace/dynatrace-operator/test/mocks/cmd/manager"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCsiCommandBuilder(t *testing.T) {
	t.Run("build command", func(t *testing.T) {
		builder := NewCsiProvisionerCommandBuilder()
		csiCommand := builder.Build()

		assert.NotNil(t, csiCommand)
		assert.Equal(t, use, csiCommand.Use)
		assert.NotNil(t, csiCommand.RunE)
	})
	t.Run("set config provider", func(t *testing.T) {
		builder := NewCsiProvisionerCommandBuilder()

		assert.NotNil(t, builder)

		expectedProvider := &configmock.Provider{}
		builder = builder.SetConfigProvider(expectedProvider)

		assert.Equal(t, expectedProvider, builder.configProvider)
	})
	t.Run("set manager provider", func(t *testing.T) {
		expectedProvider := managermock.NewProvider(t)
		builder := NewCsiProvisionerCommandBuilder().setManagerProvider(expectedProvider)

		assert.Equal(t, expectedProvider, builder.managerProvider)
	})
	t.Run("set namespace", func(t *testing.T) {
		builder := NewCsiProvisionerCommandBuilder().SetNamespace("namespace")

		assert.Equal(t, "namespace", builder.namespace)
	})
	t.Run("set filesystem", func(t *testing.T) {
		expectedFs := afero.NewMemMapFs()
		builder := NewCsiProvisionerCommandBuilder()

		assert.Equal(t, afero.NewOsFs(), builder.getFilesystem())

		builder = builder.setFilesystem(expectedFs)

		assert.Equal(t, expectedFs, builder.getFilesystem())
	})
	t.Run("set csi options", func(t *testing.T) {
		expectedOptions := dtcsi.CSIOptions{
			RootDir: dtcsi.DataPath,
		}
		builder := NewCsiProvisionerCommandBuilder().
			setCsiOptions(expectedOptions)

		assert.Equal(t, expectedOptions, builder.getCsiOptions())
	})
}

func TestCreateCsiRootPath(t *testing.T) {
	memFs := afero.NewMemMapFs()
	err := createCsiDataPath(memFs)

	require.NoError(t, err)

	exists, err := afero.Exists(memFs, dtcsi.DataPath)

	assert.True(t, exists)
	require.NoError(t, err)

	stat, err := memFs.Stat(dtcsi.DataPath)

	require.NoError(t, err)
	assert.Equal(t, fs.FileMode(0770), stat.Mode()&fs.FileMode(0770))
	assert.True(t, stat.IsDir())
}

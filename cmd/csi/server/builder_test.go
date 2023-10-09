package server

import (
	"io/fs"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/cmd/config"
	cmdManager "github.com/Dynatrace/dynatrace-operator/cmd/manager"
	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestCsiCommandBuilder(t *testing.T) {
	t.Run("build command", func(t *testing.T) {
		builder := NewCsiServerCommandBuilder()
		csiCommand := builder.Build()

		assert.NotNil(t, csiCommand)
		assert.Equal(t, use, csiCommand.Use)
		assert.NotNil(t, csiCommand.RunE)
	})
	t.Run("set config provider", func(t *testing.T) {
		builder := NewCsiServerCommandBuilder()

		assert.NotNil(t, builder)

		expectedProvider := &config.MockProvider{}
		builder = builder.SetConfigProvider(expectedProvider)

		assert.Equal(t, expectedProvider, builder.configProvider)
	})
	t.Run("set manager provider", func(t *testing.T) {
		expectedProvider := &cmdManager.MockProvider{}
		builder := NewCsiServerCommandBuilder().setManagerProvider(expectedProvider)

		assert.Equal(t, expectedProvider, builder.managerProvider)
	})
	t.Run("set namespace", func(t *testing.T) {
		builder := NewCsiServerCommandBuilder().SetNamespace("namespace")

		assert.Equal(t, "namespace", builder.namespace)
	})
	t.Run("set filesystem", func(t *testing.T) {
		expectedFs := afero.NewMemMapFs()
		builder := NewCsiServerCommandBuilder()

		assert.Equal(t, afero.NewOsFs(), builder.getFilesystem())

		builder = builder.setFilesystem(expectedFs)

		assert.Equal(t, expectedFs, builder.getFilesystem())
	})
	t.Run("set csi options", func(t *testing.T) {
		expectedOptions := dtcsi.CSIOptions{
			NodeId:   "test-node-id",
			Endpoint: "test-endpoint",
			RootDir:  dtcsi.DataPath,
		}
		builder := NewCsiServerCommandBuilder().
			setCsiOptions(expectedOptions)

		assert.Equal(t, expectedOptions, builder.getCsiOptions())
	})
}

func TestCreateCsiRootPath(t *testing.T) {
	memFs := afero.NewMemMapFs()
	err := createCsiDataPath(memFs)

	assert.NoError(t, err)

	exists, err := afero.Exists(memFs, dtcsi.DataPath)

	assert.True(t, exists)
	assert.NoError(t, err)

	stat, err := memFs.Stat(dtcsi.DataPath)

	assert.NoError(t, err)
	assert.Equal(t, fs.FileMode(0770), stat.Mode()&fs.FileMode(0770))
	assert.True(t, stat.IsDir())
}

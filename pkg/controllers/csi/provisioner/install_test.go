package csiprovisioner

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetTargetDir(t *testing.T) {
	t.Run("version set => folder is the version", func(t *testing.T) {
		prov := createProvisioner(t)
		dk := createDynaKubeWithVersion(t)

		targetDir, err := prov.getTargetDir(*dk)
		require.NoError(t, err)
		require.Contains(t, targetDir, dk.CodeModulesVersion())
	})

	t.Run("image set => folder is the base64 of the imageURI", func(t *testing.T) {
		prov := createProvisioner(t)
		dk := createDynaKubeWithImage(t)

		expectedDir := base64.StdEncoding.EncodeToString([]byte(dk.CodeModulesImage()))
		targetDir, err := prov.getTargetDir(*dk)
		require.NoError(t, err)
		require.Contains(t, targetDir, expectedDir)
	})

	t.Run("nothing set => error (shouldn't be possible in real life)", func(t *testing.T) {
		prov := createProvisioner(t)
		dk := createDynaKubeBase(t)

		targetDir, err := prov.getTargetDir(*dk)
		require.Error(t, err)
		require.Empty(t, targetDir)
	})
}

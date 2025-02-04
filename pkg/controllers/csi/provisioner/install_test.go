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

		targetDir := prov.getTargetDir(*dk)
		require.Contains(t, targetDir, dk.OneAgent().GetCodeModulesVersion())
	})

	t.Run("image set => folder is the base64 of the imageURI", func(t *testing.T) {
		prov := createProvisioner(t)
		dk := createDynaKubeWithImage(t)

		expectedDir := base64.StdEncoding.EncodeToString([]byte(dk.OneAgent().GetCodeModulesImage()))
		targetDir := prov.getTargetDir(*dk)
		require.Contains(t, targetDir, expectedDir)
	})
}

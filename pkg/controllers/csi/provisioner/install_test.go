// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

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

		targetDir := prov.getTargetDir(dk)
		require.Contains(t, targetDir, dk.OneAgent().GetCodeModulesVersion())
	})

	t.Run("image set => folder is the base64 of the imageURI", func(t *testing.T) {
		prov := createProvisioner(t)
		dk := createDynaKubeWithImage(t)

		expectedDir := base64.StdEncoding.EncodeToString([]byte(dk.OneAgent().GetCodeModulesImage()))
		targetDir := prov.getTargetDir(dk)
		require.Contains(t, targetDir, expectedDir)
	})

	t.Run("version with double-dot => double-dot stripped from dir", func(t *testing.T) {
		prov := createProvisioner(t)
		dk := createDynaKubeWithVersion(t)
		dk.Status.CodeModules.Version = "1.0../evil"

		targetDir := prov.getTargetDir(dk)
		require.NotContains(t, targetDir, "..")
	})

	t.Run("version with colon => colon stripped from dir", func(t *testing.T) {
		prov := createProvisioner(t)
		dk := createDynaKubeWithVersion(t)
		dk.Status.CodeModules.Version = "version:1.0"

		targetDir := prov.getTargetDir(dk)
		require.NotContains(t, targetDir, ":")
	})

	t.Run("version with comma => comma stripped from dir", func(t *testing.T) {
		prov := createProvisioner(t)
		dk := createDynaKubeWithVersion(t)
		dk.Status.CodeModules.Version = "v1,0"

		targetDir := prov.getTargetDir(dk)
		require.NotContains(t, targetDir, ",")
	})

	t.Run("version disguising root separator => dir name is empty", func(t *testing.T) {
		prov := createProvisioner(t)
		dk := createDynaKubeWithVersion(t)
		dk.Status.CodeModules.Version = "../"

		targetDir := prov.getTargetDir(dk)
		require.Equal(t, prov.path.AgentSharedBinaryDirBase(), targetDir)
	})
}

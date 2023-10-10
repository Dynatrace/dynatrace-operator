package metadata

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPathResolver(t *testing.T) {
	rootDir := "/testroot/tmp"
	tenantUUID := "asj23443"

	pathResolver := PathResolver{RootDir: rootDir}
	fakeEnv := filepath.Join(rootDir, tenantUUID)
	fakeVolume := "csi-sdf3ijiji3jldisomeid"
	agentRunDirForVolume := filepath.Join(fakeEnv, "run", fakeVolume)

	assert.Equal(t, fakeEnv, pathResolver.TenantDir(tenantUUID))
	assert.Equal(t, filepath.Join(fakeEnv, "bin"), pathResolver.AgentBinaryDir(tenantUUID))
	assert.Equal(t, filepath.Join(fakeEnv, "bin", "v1"), pathResolver.AgentBinaryDirForVersion(tenantUUID, "v1"))
	assert.Equal(t, filepath.Join(fakeEnv, "run"), pathResolver.AgentRunDir(tenantUUID))
	assert.Equal(t, agentRunDirForVolume, pathResolver.AgentRunDirForVolume(tenantUUID, fakeVolume))
	assert.Equal(t, filepath.Join(agentRunDirForVolume, "mapped"), pathResolver.OverlayMappedDir(tenantUUID, fakeVolume))
	assert.Equal(t, filepath.Join(agentRunDirForVolume, "var"), pathResolver.OverlayVarDir(tenantUUID, fakeVolume))
	assert.Equal(t, filepath.Join(agentRunDirForVolume, "work"), pathResolver.OverlayWorkDir(tenantUUID, fakeVolume))
}

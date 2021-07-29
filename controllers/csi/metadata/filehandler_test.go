package metadata

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilePathHandler(t *testing.T) {
	rootDir := "/testroot/tmp"
	tenantUUID := "asj23443"

	fph := FilePathHandler{RootDir: rootDir}
	fakeEnv := filepath.Join(rootDir, tenantUUID)
	fakeVolume := "csi-sdf3ijiji3jldisomeid"
	agentRunDirForVolume := filepath.Join(fakeEnv, "run", fakeVolume)

	assert.Equal(t, fakeEnv, fph.EnvDir(tenantUUID))
	assert.Equal(t, filepath.Join(fakeEnv, "bin"), fph.AgentBinaryDir(tenantUUID))
	assert.Equal(t, filepath.Join(fakeEnv, "bin", "v1"), fph.AgentBinaryDirForVersion(tenantUUID, "v1"))
	assert.Equal(t, filepath.Join(fakeEnv, "run"), fph.AgentRunDir(tenantUUID))
	assert.Equal(t, agentRunDirForVolume, fph.AgentRunDirForVolume(tenantUUID, fakeVolume))
	assert.Equal(t, filepath.Join(agentRunDirForVolume, "mapped"), fph.OverlayMappedDir(tenantUUID, fakeVolume))
	assert.Equal(t, filepath.Join(agentRunDirForVolume, "var"), fph.OverlayVarDir(tenantUUID, fakeVolume))
	assert.Equal(t, filepath.Join(agentRunDirForVolume, "work"), fph.OverlayWorkDir(tenantUUID, fakeVolume))
}

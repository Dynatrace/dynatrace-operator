package storage

import (
	"path/filepath"

	dtcsi "github.com/Dynatrace/dynatrace-operator/controllers/csi"
)

type FilePathHandler struct {
	RootDir string
}

func (fh FilePathHandler) EnvDir(tenantUUID string) string {
	return filepath.Join(fh.RootDir, tenantUUID)
}

func (fh FilePathHandler) AgentBinaryDir(tenantUUID string) string {
	return filepath.Join(fh.EnvDir(tenantUUID), dtcsi.AgentBinaryDir)
}

func (fh FilePathHandler) AgentBinaryDirForVersion(tenantUUID string, version string) string {
	return filepath.Join(fh.AgentBinaryDir(tenantUUID), version)
}

func (fh FilePathHandler) AgentRunDir(tenantUUID string) string {
	return filepath.Join(fh.EnvDir(tenantUUID), dtcsi.AgentRunDir)
}

func (fh FilePathHandler) AgentRunDirForVolume(tenantUUID string, volumeId string) string {
	return filepath.Join(fh.AgentRunDir(tenantUUID), volumeId)
}

func (fh FilePathHandler) OverlayMappedDir(tenantUUID string, volumeId string) string {
	return filepath.Join(fh.AgentRunDirForVolume(tenantUUID, volumeId), dtcsi.OverlayMappedDirPath)
}

func (fh FilePathHandler) OverlayVarDir(tenantUUID string, volumeId string) string {
	return filepath.Join(fh.AgentRunDirForVolume(tenantUUID, volumeId), dtcsi.OverlayVarDirPath)
}

func (fh FilePathHandler) OverlayWorkDir(tenantUUID string, volumeId string) string {
	return filepath.Join(fh.AgentRunDirForVolume(tenantUUID, volumeId), dtcsi.OverlayWorkDirPath)
}

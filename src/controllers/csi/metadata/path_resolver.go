package metadata

import (
	"path/filepath"

	dtcsi "github.com/Dynatrace/dynatrace-operator/src/controllers/csi"
)

type PathResolver struct {
	RootDir string
}

func (pr PathResolver) EnvDir(tenantUUID string) string {
	return filepath.Join(pr.RootDir, tenantUUID)
}

func (pr PathResolver) OsAgentDir(tenantUUID string) string {
	return filepath.Join(pr.EnvDir(tenantUUID), "osagent")
}

func (pr PathResolver) AgentBinaryDir(tenantUUID string) string {
	return filepath.Join(pr.EnvDir(tenantUUID), dtcsi.AgentBinaryDir)
}

func (pr PathResolver) ImageCertPath(tenantUUID string) string {
	return filepath.Join(pr.EnvDir(tenantUUID), "ca.crt")
}

func (pr PathResolver) AgentProcessModuleConfigForVersion(tenantUUID string, version string) string {
	return filepath.Join(pr.AgentBinaryDirForVersion(tenantUUID, version), "agent", "conf", "ruxitagentproc.conf")
}

func (pr PathResolver) SourceAgentProcessModuleConfigForVersion(tenantUUID string, version string) string {
	return filepath.Join(pr.AgentBinaryDirForVersion(tenantUUID, version), "agent", "conf", "_ruxitagentproc.conf")
}

func (pr PathResolver) AgentRuxitProcResponseCache(tenantUUID string) string {
	return filepath.Join(pr.EnvDir(tenantUUID), "revision.json")
}

func (pr PathResolver) AgentBinaryDirForVersion(tenantUUID string, version string) string {
	return filepath.Join(pr.AgentBinaryDir(tenantUUID), version)
}

func (pr PathResolver) AgentSharedBinaryDirBase() string {
	return filepath.Join(pr.RootDir, "codemodules")
}

func (pr PathResolver) AgentSharedBinaryDirForDigest(digest string) string {
	return filepath.Join(pr.AgentSharedBinaryDirBase(), digest)
}

func (pr PathResolver) AgentConfigDir(tenantUUID string) string {
	return filepath.Join(pr.EnvDir(tenantUUID), "config")
}

func (pr PathResolver) InnerAgentBinaryDirForSymlinkForVersion(tenantUUID string, version string) string {
	return filepath.Join(pr.AgentBinaryDirForVersion(tenantUUID, version), "agent", "bin", "current")
}

func (pr PathResolver) AgentRunDir(tenantUUID string) string {
	return filepath.Join(pr.EnvDir(tenantUUID), dtcsi.AgentRunDir)
}

func (pr PathResolver) AgentRunDirForVolume(tenantUUID string, volumeId string) string {
	return filepath.Join(pr.AgentRunDir(tenantUUID), volumeId)
}

func (pr PathResolver) OverlayMappedDir(tenantUUID string, volumeId string) string {
	return filepath.Join(pr.AgentRunDirForVolume(tenantUUID, volumeId), dtcsi.OverlayMappedDirPath)
}

func (pr PathResolver) OverlayVarDir(tenantUUID string, volumeId string) string {
	return filepath.Join(pr.AgentRunDirForVolume(tenantUUID, volumeId), dtcsi.OverlayVarDirPath)
}

func (pr PathResolver) OverlayWorkDir(tenantUUID string, volumeId string) string {
	return filepath.Join(pr.AgentRunDirForVolume(tenantUUID, volumeId), dtcsi.OverlayWorkDirPath)
}

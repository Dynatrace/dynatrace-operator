package metadata

import (
	"path/filepath"

	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/processmoduleconfig"
)

type PathResolver struct {
	RootDir string
}

func (pr PathResolver) TenantDir(tenantUUID string) string {
	return filepath.Join(pr.RootDir, tenantUUID)
}

func (pr PathResolver) OsAgentDir(tenantUUID string) string {
	return filepath.Join(pr.TenantDir(tenantUUID), "osagent")
}

// Deprecated
func (pr PathResolver) AgentBinaryDir(tenantUUID string) string {
	return filepath.Join(pr.TenantDir(tenantUUID), dtcsi.AgentBinaryDir)
}

// Deprecated
func (pr PathResolver) AgentBinaryDirForVersion(tenantUUID string, version string) string {
	return filepath.Join(pr.AgentBinaryDir(tenantUUID), version)
}

func (pr PathResolver) AgentSharedBinaryDirBase() string {
	return filepath.Join(pr.RootDir, dtcsi.SharedAgentBinDir)
}

func (pr PathResolver) LatestAgentBinaryForDynaKube(dynakubeName string) string {
	return filepath.Join(pr.RootDir, dynakubeName, "latest-codemodule")
}


func (pr PathResolver) AgentTempUnzipRootDir() string {
	return filepath.Join(pr.RootDir, "tmp_zip")
}

func (pr PathResolver) AgentTempUnzipDir() string {
	return filepath.Join(pr.AgentTempUnzipRootDir(), "opt", "dynatrace", "oneagent")
}

func (pr PathResolver) AgentSharedBinaryDirForAgent(versionOrDigest string) string {
	return filepath.Join(pr.AgentSharedBinaryDirBase(), versionOrDigest)
}

func (pr PathResolver) AgentConfigDir(tenantUUID string, dynakubeName string) string {
	return filepath.Join(pr.TenantDir(tenantUUID), dynakubeName, dtcsi.SharedAgentConfigDir)
}

func (pr PathResolver) AgentSharedRuxitAgentProcConf(tenantUUID, dynakubeName string) string {
	return filepath.Join(pr.AgentConfigDir(tenantUUID, dynakubeName), processmoduleconfig.RuxitAgentProcPath)
}

func (pr PathResolver) OverlayVarRuxitAgentProcConf(tenantUUID, volumeId string) string {
	return filepath.Join(pr.OverlayVarDir(tenantUUID, volumeId), processmoduleconfig.RuxitAgentProcPath)
}

func (pr PathResolver) AgentRunDir(tenantUUID string) string {
	return filepath.Join(pr.TenantDir(tenantUUID), dtcsi.AgentRunDir)
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

package metadata

import (
	"path/filepath"

	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/processmoduleconfig"
)

type PathResolver struct {
	RootDir string
}

func (pr PathResolver) DynaKubeDir(dynakubeName string) string {
	return filepath.Join(pr.RootDir, dynakubeName)
}

func (pr PathResolver) OsAgentDir(dynakubeName string) string {
	return filepath.Join(pr.DynaKubeDir(dynakubeName), "osagent")
}

// Deprecated
func (pr PathResolver) AgentBinaryDir(tenantUUID string) string {
	return filepath.Join(pr.DynaKubeDir(tenantUUID), dtcsi.AgentBinaryDir)
}

// Deprecated
func (pr PathResolver) AgentBinaryDirForVersion(tenantUUID string, version string) string {
	return filepath.Join(pr.AgentBinaryDir(tenantUUID), version)
}

func (pr PathResolver) AgentSharedBinaryDirBase() string {
	return filepath.Join(pr.RootDir, dtcsi.SharedAgentBinDir)
}

func (pr PathResolver) LatestAgentBinaryForDynaKube(dynakubeName string) string {
	return filepath.Join(pr.DynaKubeDir(dynakubeName), "latest-codemodule")
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

func (pr PathResolver) AgentConfigDir(dynakubeName string) string {
	return filepath.Join(pr.DynaKubeDir(dynakubeName), dtcsi.SharedAgentConfigDir)
}

func (pr PathResolver) AgentSharedRuxitAgentProcConf(dynakubeName string) string {
	return filepath.Join(pr.AgentConfigDir(dynakubeName), processmoduleconfig.RuxitAgentProcPath)
}

func (pr PathResolver) OverlayVarRuxitAgentProcConf(dynakubeName, volumeId string) string {
	return filepath.Join(pr.OverlayVarDir(dynakubeName, volumeId), processmoduleconfig.RuxitAgentProcPath)
}

func (pr PathResolver) AgentRunDir(dynakubeName string) string {
	return filepath.Join(pr.DynaKubeDir(dynakubeName), dtcsi.AgentRunDir)
}

func (pr PathResolver) AgentRunDirForVolume(dynakubeName string, volumeId string) string {
	return filepath.Join(pr.AgentRunDir(dynakubeName), volumeId)
}

func (pr PathResolver) OverlayMappedDir(dynakubeName string, volumeId string) string {
	return filepath.Join(pr.AgentRunDirForVolume(dynakubeName, volumeId), dtcsi.OverlayMappedDirPath)
}

func (pr PathResolver) OverlayVarDir(dynakubeName string, volumeId string) string {
	return filepath.Join(pr.AgentRunDirForVolume(dynakubeName, volumeId), dtcsi.OverlayVarDirPath)
}

func (pr PathResolver) OverlayWorkDir(dynakubeName string, volumeId string) string {
	return filepath.Join(pr.AgentRunDirForVolume(dynakubeName, volumeId), dtcsi.OverlayWorkDirPath)
}

// Deprecated kept for future migration/cleanup
func (pr PathResolver) OldAgentConfigDir(tenantUUID string, dynakubeName string) string {
	return filepath.Join(pr.DynaKubeDir(tenantUUID), dynakubeName, dtcsi.SharedAgentConfigDir)
}

// Deprecated kept for future migration/cleanup
func (pr PathResolver) OldAgentSharedRuxitAgentProcConf(tenantUUID, dynakubeName string) string {
	return filepath.Join(pr.OldAgentConfigDir(tenantUUID, dynakubeName), processmoduleconfig.RuxitAgentProcPath)
}

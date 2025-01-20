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

func (pr PathResolver) AgentSharedBinaryDirBase() string {
	return filepath.Join(pr.RootDir, dtcsi.SharedAgentBinDir)
}

func (pr PathResolver) AgentSharedBinaryDirForAgent(versionOrDigest string) string {
	return filepath.Join(pr.AgentSharedBinaryDirBase(), versionOrDigest)
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

func (pr PathResolver) AgentConfigDir(dynakubeName string) string {
	return filepath.Join(pr.DynaKubeDir(dynakubeName), dtcsi.SharedAgentConfigDir)
}

func (pr PathResolver) AgentSharedRuxitAgentProcConf(dynakubeName string) string {
	return filepath.Join(pr.AgentConfigDir(dynakubeName), processmoduleconfig.RuxitAgentProcPath)
}

func (pr PathResolver) OverlayVarRuxitAgentProcConf(volumeID string) string {
	return filepath.Join(pr.AppMountVarDir(volumeID), processmoduleconfig.RuxitAgentProcPath)
}

func (pr PathResolver) OverlayVarPodInfo(volumeID string) string {
	return filepath.Join(pr.AppMountVarDir(volumeID), "pod-info")
}

// AppMountsBaseDir replaces the AgentRunDir, the base directory where all the volumes for the app-mounts are stored
func (pr PathResolver) AppMountsBaseDir() string {
	return filepath.Join(pr.RootDir, dtcsi.SharedAppMountsDir)
}

// AppMountForID replaces AgentRunDirForVolume, the directory where a given app-mount volume is stored
func (pr PathResolver) AppMountForID(volumeID string) string {
	return filepath.Join(pr.AppMountsBaseDir(), volumeID)
}

// AppMountForDK is a directory where a given app-mount volume is stored under a certain dynakube
func (pr PathResolver) AppMountForDK(dkName string) string {
	return filepath.Join(pr.RootDir, dkName, dtcsi.SharedAppMountsDir)
}

// AppMountMappedDir replaces OverlayMappedDir, the directory where the overlay layers combine into
func (pr PathResolver) AppMountMappedDir(volumeID string) string {
	return filepath.Join(pr.AppMountForID(volumeID), dtcsi.OverlayMappedDirPath)
}

// AppMountVarDir replaces OverlayVarDir, the directory where the container using the volume writes
func (pr PathResolver) AppMountVarDir(volumeID string) string {
	return filepath.Join(pr.AppMountForID(volumeID), dtcsi.OverlayVarDirPath)
}

// AppMountWorkDir replaces OverlayWorkDir, the directory that is necessary for overlayFS to work
func (pr PathResolver) AppMountWorkDir(volumeID string) string {
	return filepath.Join(pr.AppMountForID(volumeID), dtcsi.OverlayWorkDirPath)
}

func (pr PathResolver) AppMountPodInfoDir(dkName, podNamespace, podName string) string {
	return filepath.Join(pr.AppMountForDK(dkName), podNamespace, podName)
}

// Deprecated kept for future migration/cleanup
func (pr PathResolver) AgentRunDir(dynakubeName string) string {
	return filepath.Join(pr.DynaKubeDir(dynakubeName), dtcsi.AgentRunDir)
}

// Deprecated kept for future migration/cleanup
func (pr PathResolver) AgentRunDirForVolume(dynakubeName string, volumeId string) string {
	return filepath.Join(pr.AgentRunDir(dynakubeName), volumeId)
}

// Deprecated kept for future migration/cleanup
func (pr PathResolver) OverlayMappedDir(dynakubeName string, volumeId string) string {
	return filepath.Join(pr.AgentRunDirForVolume(dynakubeName, volumeId), dtcsi.OverlayMappedDirPath)
}

// Deprecated kept for future migration/cleanup
func (pr PathResolver) OverlayVarDir(dynakubeName string, volumeId string) string {
	return filepath.Join(pr.AgentRunDirForVolume(dynakubeName, volumeId), dtcsi.OverlayVarDirPath)
}

// Deprecated kept for future migration/cleanup
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

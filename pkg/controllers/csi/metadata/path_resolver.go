package metadata

import (
	"path/filepath"

	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
)

type PathResolver struct {
	RootDir string
}

// For the server

func (pr PathResolver) OsMountDir() string {
	return filepath.Join(pr.RootDir, "os-storage")
}

func (pr PathResolver) AppMountsBaseDir() string {
	return filepath.Join(pr.RootDir, "app-mounts")
}

func (pr PathResolver) AppMountsDir(namespace, pod, volumeID string) string {
	return filepath.Join(pr.AppMountsBaseDir(), namespace, pod, volumeID)
}

func (pr PathResolver) OverlayMappedDir(namespace, pod, volumeID string) string {
	return filepath.Join(pr.AppMountsDir(namespace, pod, volumeID), dtcsi.OverlayMappedDirPath)
}

func (pr PathResolver) OverlayVarDir(namespace, pod, volumeID string) string {
	return filepath.Join(pr.AppMountsDir(namespace, pod, volumeID), dtcsi.OverlayVarDirPath)
}

func (pr PathResolver) OverlayWorkDir(namespace, pod, volumeID string) string {
	return filepath.Join(pr.AppMountsDir(namespace, pod, volumeID), dtcsi.OverlayWorkDirPath)
}

// For the provisioner

func (pr PathResolver) SharedCodeModulesBaseDir() string {
	return filepath.Join(pr.RootDir, dtcsi.SharedAgentBinDir)
}

func (pr PathResolver) SharedCodeModulesDirForVersion(versionOrImageURI string) string {
	return filepath.Join(pr.SharedCodeModulesBaseDir(), versionOrImageURI)
}

func (pr PathResolver) AgentTempUnzipRootDir() string {
	return filepath.Join(pr.RootDir, "tmp_zip")
}

func (pr PathResolver) AgentTempUnzipDir() string {
	return filepath.Join(pr.AgentTempUnzipRootDir(), "opt", "dynatrace", "oneagent")
}


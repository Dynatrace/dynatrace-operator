package csigc

import (
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

func (gc *CSIGarbageCollector) runSharedBinaryGarbageCollection() error {
	imageDirs, err := gc.getSharedBinDirs()
	if err != nil {
		return err
	}

	if len(imageDirs) == 0 {
		log.Info("no shared binary dirs on node")

		return nil
	}

	binsToDelete, err := gc.collectUnusedAgentBins(imageDirs)
	if err != nil {
		return err
	}

	if len(binsToDelete) == 0 {
		log.Info("no shared binary dirs to delete on the node")

		return nil
	}

	return deleteSharedBinDirs(gc.fs, binsToDelete)
}

func (gc *CSIGarbageCollector) getSharedBinDirs() ([]os.FileInfo, error) {
	imageDirs, err := afero.Afero{Fs: gc.fs}.ReadDir(gc.path.AgentSharedBinaryDirBase())
	if os.IsNotExist(err) {
		return nil, nil
	}

	if err != nil {
		log.Info("failed to read shared image directory")

		return nil, errors.WithStack(err)
	}

	return imageDirs, nil
}

func (gc *CSIGarbageCollector) collectUnusedAgentBins(imageDirs []os.FileInfo) ([]string, error) {
	var toDelete []string

	usedAgentBinPaths, err := gc.getUsedAgentBinPaths()
	if err != nil {
		log.Info("failed to get the used agent bin paths")

		return nil, err
	}

	codeModuleAgentBinPaths, err := gc.getCodeModuleAgentBinPaths()
	if err != nil {
		log.Info("failed to get CodeModule bin paths")

		return nil, err
	}

	for _, imageDir := range imageDirs {
		agentBinPath := gc.path.AgentSharedBinaryDirForAgent(imageDir.Name())

		if !codeModuleAgentBinPaths[agentBinPath] && !usedAgentBinPaths[agentBinPath] {
			toDelete = append(toDelete, agentBinPath)
		}
	}

	return toDelete, nil
}

// Returns a map with all agent bin paths based on existing TenantConfig.DownloadedCodeModuleVersion
// (which is the latest downloaded CodeModule version from the tenant)
func (gc *CSIGarbageCollector) getUsedAgentBinPaths() (map[string]bool, error) {
	tenantConfigs, err := gc.db.ReadTenantConfigs()
	if err != nil {
		return nil, err
	}

	latestCodeModuleVersions := make(map[string]bool)

	for _, tenantConfig := range tenantConfigs {
		agentBinPath := gc.path.AgentSharedBinaryDirForAgent(tenantConfig.DownloadedCodeModuleVersion)
		latestCodeModuleVersions[agentBinPath] = true
	}

	return latestCodeModuleVersions, nil
}

// Returns a map with all agent bin paths based on existing CodeModule entries
func (gc *CSIGarbageCollector) getCodeModuleAgentBinPaths() (map[string]bool, error) {
	codeModules, err := gc.db.ReadCodeModules()
	if err != nil {
		return nil, err
	}

	codeModuleBinPaths := make(map[string]bool)

	for _, codeModule := range codeModules {
		codeModuleBinPaths[codeModule.Location] = true
	}

	return codeModuleBinPaths, nil
}

func deleteSharedBinDirs(fs afero.Fs, imageDirs []string) error {
	for _, dir := range imageDirs {
		log.Info("deleting shared image dir", "dir", dir)

		err := fs.RemoveAll(dir)
		if err != nil {
			log.Info("failed to delete image cache", "dir", dir)

			return errors.WithStack(err)
		}
	}

	return nil
}

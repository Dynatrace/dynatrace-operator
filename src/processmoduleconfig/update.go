package processmoduleconfig

import (
	"io"
	"os"
	"path/filepath"

	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/spf13/afero"
)

var (
	ruxitAgentProcPath       = filepath.Join("agent", "conf", "ruxitagentproc.conf")
	sourceRuxitAgentProcPath = filepath.Join("agent", "conf", "_ruxitagentproc.conf")
)

func UpdateProcessModuleConfig(fs afero.Fs, targetDir string, processModuleConfig *dtclient.ProcessModuleConfig) error {
	if processModuleConfig != nil {
		log.Info("updating ruxitagentproc.conf", "targetDir", targetDir)
		usedProcessModuleConfigPath := filepath.Join(targetDir, ruxitAgentProcPath)
		sourceProcessModuleConfigPath := filepath.Join(targetDir, sourceRuxitAgentProcPath)
		if err := checkProcessModuleConfigCopy(fs, sourceProcessModuleConfigPath, usedProcessModuleConfigPath); err != nil {
			return err
		}
		return Update(fs, sourceProcessModuleConfigPath, usedProcessModuleConfigPath, processModuleConfig.ToMap())
	}
	log.Info("no changes to ruxitagentproc.conf, skipping update")
	return nil
}

// checkProcessModuleConfigCopy checks if we already made a copy of the original ruxitagentproc.conf file.
// After the initial installation of a version we copy the ruxitagentproc.conf to _ruxitagentproc.conf, and we use the _ruxitagentproc.conf + the api response to re-create the ruxitagentproc.conf
// so it`s easier to update
func checkProcessModuleConfigCopy(fs afero.Fs, sourcePath, destPath string) error {
	if _, err := fs.Open(sourcePath); os.IsNotExist(err) {
		log.Info("saving original ruxitagentproc.conf to _ruxitagentproc.conf")
		fileInfo, err := fs.Stat(destPath)
		if err != nil {
			return err
		}

		sourceProcessModuleConfigFile, err := fs.OpenFile(sourcePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fileInfo.Mode())
		if err != nil {
			return err
		}

		usedProcessModuleConfigFile, err := fs.Open(destPath)
		if err != nil {
			return err
		}

		_, err = io.Copy(sourceProcessModuleConfigFile, usedProcessModuleConfigFile)
		if err != nil {
			if err := sourceProcessModuleConfigFile.Close(); err != nil {
				log.Error(err, "failed to close sourceProcessModuleConfigFile")
			}
			if err := usedProcessModuleConfigFile.Close(); err != nil {
				log.Error(err, "failed to close usedProcessModuleConfigFile")
			}
			return err
		}
		if err = sourceProcessModuleConfigFile.Close(); err != nil {
			return err
		}
		return usedProcessModuleConfigFile.Close()
	}
	return nil
}

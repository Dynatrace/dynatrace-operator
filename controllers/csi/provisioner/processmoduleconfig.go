package csiprovisioner

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"os"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/controllers/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/Dynatrace/dynatrace-operator/processmoduleconfig"
)

type processModuleConfigCache struct {
	*dtclient.ProcessModuleConfig
	Hash string `json:"hash"`
}

func newProcessModuleConfigCache(pmc *dtclient.ProcessModuleConfig) *processModuleConfigCache {
	if pmc == nil {
		pmc = &dtclient.ProcessModuleConfig{}
	}
	hash, err := kubeobjects.GenerateHash(pmc)
	if err != nil {
		return nil
	}
	return &processModuleConfigCache{
		pmc,
		hash,
	}
}

// getProcessModuleConfig gets the latest `RuxitProcResponse`, it can come from the tenant if we don't have the latest revision saved locally,
// otherwise we use the locally cached response
func (r *OneAgentProvisioner) getProcessModuleConfig(dtc dtclient.Client, tenantUUID string) (*dtclient.ProcessModuleConfig, string, error) {
	var storedHash string
	storedProcessModuleConfig, err := r.readProcessModuleConfigCache(tenantUUID)
	if os.IsNotExist(err) {
		latestProcessModuleConfig, err := dtc.GetProcessModuleConfig(0)
		if err != nil {
			return nil, storedHash, err
		}
		return latestProcessModuleConfig, storedHash, nil
	} else if err != nil {
		return nil, storedHash, err
	}
	storedHash = storedProcessModuleConfig.Hash
	latestProcessModuleConfig, err := dtc.GetProcessModuleConfig(storedProcessModuleConfig.Revision)
	if err != nil {
		return nil, storedHash, err
	}
	if latestProcessModuleConfig != nil {
		return latestProcessModuleConfig, storedHash, nil
	}
	return storedProcessModuleConfig.ProcessModuleConfig, storedHash, nil
}

func (r *OneAgentProvisioner) readProcessModuleConfigCache(tenantUUID string) (*processModuleConfigCache, error) {
	var processModuleConfig processModuleConfigCache
	processModuleConfigCache, err := r.fs.Open(r.path.AgentRuxitProcResponseCache(tenantUUID))
	if err != nil {
		return nil, err
	}
	jsonBytes, err := ioutil.ReadAll(processModuleConfigCache)
	processModuleConfigCache.Close()
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(jsonBytes, &processModuleConfig); err != nil {
		return nil, err
	}

	return &processModuleConfig, nil
}

func (r *OneAgentProvisioner) writeProcessModuleConfigCache(tenantUUID string, processModuleConfig *processModuleConfigCache) error {
	processModuleConfigCache, err := r.fs.OpenFile(r.path.AgentRuxitProcResponseCache(tenantUUID), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	jsonBytes, err := json.Marshal(processModuleConfig)
	if err != nil {
		processModuleConfigCache.Close()
		return err
	}
	_, err = processModuleConfigCache.Write(jsonBytes)
	processModuleConfigCache.Close()
	return err
}

func (installAgentCfg *installAgentConfig) updateProcessModuleConfig(version, tenantUUID string, processModuleConfig *dtclient.ProcessModuleConfig) error {
	if processModuleConfig != nil {
		installAgentCfg.logger.Info("updating ruxitagentproc.conf", "agentVersion", version, "tenantUUID", tenantUUID)
		usedProcessModuleConfigPath := installAgentCfg.path.AgentProcessModuleConfigForVersion(tenantUUID, version)
		sourceProcessModuleConfigPath := installAgentCfg.path.SourceAgentProcessModuleConfigForVersion(tenantUUID, version)
		if err := installAgentCfg.checkProcessModuleConfigCopy(sourceProcessModuleConfigPath, usedProcessModuleConfigPath); err != nil {
			return err
		}
		return processmoduleconfig.Update(installAgentCfg.fs, sourceProcessModuleConfigPath, usedProcessModuleConfigPath, processModuleConfig.ToMap())
	}
	installAgentCfg.logger.Info("no changes to ruxitagentproc.conf, skipping update")
	return nil
}

// checkProcessModuleConfigCopy checks if we already made a copy of the original ruxitagentproc.conf file.
// After the initial install of a version we copy the ruxitagentproc.conf to _ruxitagentproc.conf and we use the _ruxitagentproc.conf + the api response to re-create the ruxitagentproc.conf
// so its easier to update
func (installAgentCfg *installAgentConfig) checkProcessModuleConfigCopy(sourcePath, destPath string) error {
	if _, err := installAgentCfg.fs.Open(sourcePath); os.IsNotExist(err) {
		fileInfo, err := installAgentCfg.fs.Stat(destPath)
		if err != nil {
			return err
		}

		sourceProcessModuleConfigFile, err := installAgentCfg.fs.OpenFile(sourcePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fileInfo.Mode())
		if err != nil {
			return err
		}

		usedProcessModuleConfigFile, err := installAgentCfg.fs.Open(destPath)
		if err != nil {
			return err
		}
		_, err = io.Copy(sourceProcessModuleConfigFile, usedProcessModuleConfigFile)
		if err != nil {
			sourceProcessModuleConfigFile.Close()
			usedProcessModuleConfigFile.Close()
			return err
		}
		if err = sourceProcessModuleConfigFile.Close(); err != nil {
			return err
		}
		return usedProcessModuleConfigFile.Close()
	}
	return nil
}

func addHostGroup(dk *dynatracev1beta1.DynaKube, pmc *dtclient.ProcessModuleConfig) *dtclient.ProcessModuleConfig {
	hostGroup := dk.HostGroup()
	if hostGroup == "" {
		return pmc
	}
	pmc.Properties = append(pmc.Properties, dtclient.ProcessModuleProperty{Section: "general", Key: "hostGroup", Value: hostGroup})
	return pmc
}

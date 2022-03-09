package csiprovisioner

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
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
func (provisioner *OneAgentProvisioner) getProcessModuleConfig(dtc dtclient.Client, tenantUUID string) (*dtclient.ProcessModuleConfig, string, error) {
	var storedHash string
	storedProcessModuleConfig, err := provisioner.readProcessModuleConfigCache(tenantUUID)
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
	if latestProcessModuleConfig != nil && !latestProcessModuleConfig.IsEmpty() {
		return latestProcessModuleConfig, storedHash, nil
	}
	return storedProcessModuleConfig.ProcessModuleConfig, storedHash, nil
}

func (provisioner *OneAgentProvisioner) readProcessModuleConfigCache(tenantUUID string) (*processModuleConfigCache, error) {
	var processModuleConfig processModuleConfigCache
	processModuleConfigCache, err := provisioner.fs.Open(provisioner.path.AgentRuxitProcResponseCache(tenantUUID))
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

func (provisioner *OneAgentProvisioner) writeProcessModuleConfigCache(tenantUUID string, processModuleConfig *processModuleConfigCache) error {
	processModuleConfigCache, err := provisioner.fs.OpenFile(provisioner.path.AgentRuxitProcResponseCache(tenantUUID), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
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

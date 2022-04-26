package csiprovisioner

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient/types"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/pkg/errors"
)

type processModuleConfigCache struct {
	*types.ProcessModuleConfig
	Hash string `json:"hash"`
}

func newProcessModuleConfigCache(pmc *types.ProcessModuleConfig) *processModuleConfigCache {
	if pmc == nil {
		pmc = &types.ProcessModuleConfig{}
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
func (provisioner *OneAgentProvisioner) getProcessModuleConfig(dtc dtclient.Client, tenantUUID string) (*types.ProcessModuleConfig, string, error) {
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
	processModuleConfigCacheFile, err := provisioner.fs.Open(provisioner.path.AgentRuxitProcResponseCache(tenantUUID))
	if err != nil {
		provisioner.removeProcessModuleConfigCache(tenantUUID)
		return nil, err
	}

	jsonBytes, err := ioutil.ReadAll(processModuleConfigCacheFile)
	if err != nil {
		if err := processModuleConfigCacheFile.Close(); err != nil {
			log.Error(errors.WithStack(err), "error closing file after trying to read it")
		}
		provisioner.removeProcessModuleConfigCache(tenantUUID)
		return nil, errors.Wrapf(err, "error reading processModuleConfigCache")
	}

	if err := processModuleConfigCacheFile.Close(); err != nil {
		provisioner.removeProcessModuleConfigCache(tenantUUID)
		return nil, errors.Wrapf(err, "error closing file after reading processModuleConfigCache")
	}

	if err := json.Unmarshal(jsonBytes, &processModuleConfig); err != nil {
		provisioner.removeProcessModuleConfigCache(tenantUUID)
		return nil, errors.Wrapf(err, "error when unmarshalling processModuleConfigCache")
	}

	return &processModuleConfig, nil
}

func (provisioner *OneAgentProvisioner) writeProcessModuleConfigCache(tenantUUID string, processModuleConfig *processModuleConfigCache) error {
	processModuleConfigCacheFile, err := provisioner.fs.OpenFile(provisioner.path.AgentRuxitProcResponseCache(tenantUUID), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		provisioner.removeProcessModuleConfigCache(tenantUUID)
		return errors.Wrapf(err, "error opening processModuleConfigCache, when writing")
	}

	jsonBytes, err := json.Marshal(processModuleConfig)
	if err != nil {
		if err := processModuleConfigCacheFile.Close(); err != nil {
			log.Error(errors.WithStack(err), "error closing file after trying to unmarshal")
		}
		provisioner.removeProcessModuleConfigCache(tenantUUID)
		return errors.Wrapf(err, "error when marshaling processModuleConfigCache")
	}

	if _, err := processModuleConfigCacheFile.Write(jsonBytes); err != nil {
		provisioner.removeProcessModuleConfigCache(tenantUUID)
		return errors.Wrapf(err, "error writing processModuleConfigCache")
	}

	if err := processModuleConfigCacheFile.Close(); err != nil {
		provisioner.removeProcessModuleConfigCache(tenantUUID)
		return errors.Wrapf(err, "error closing file after writing processModuleConfigCache")
	}
	return nil
}

func (provisioner *OneAgentProvisioner) removeProcessModuleConfigCache(tenantUUID string) {
	err := provisioner.fs.Remove(provisioner.path.AgentRuxitProcResponseCache(tenantUUID))
	if err != nil {
		log.Error(errors.WithStack(err), "error removing processModuleConfigCache")
	}
}

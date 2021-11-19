package csiprovisioner

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"os"

	"github.com/Dynatrace/dynatrace-operator/conf"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
)

// getRuxitProcResponse gets the latest `RuxitProcResponse`, it can come from the tenant if we don't have the latest revision saved locally,
// otherwise we use the locally cached response
func (r *OneAgentProvisioner) getRuxitProcResponse(dtc dtclient.Client, tenantUUID string) (*dtclient.RuxitProcResponse, uint, error) {
	var lastRevision uint
	storedRuxitProcResponse, err := r.readRuxitCache(tenantUUID)
	if os.IsNotExist(err) {
		latestRuxitProcResponse, err := dtc.GetRuxitProcConf(lastRevision)
		if err != nil {
			return nil, lastRevision, err
		}
		return latestRuxitProcResponse, lastRevision, nil
	} else if err != nil {
		return nil, lastRevision, err
	}
	lastRevision = storedRuxitProcResponse.Revision
	latestRuxitProcResponse, err := dtc.GetRuxitProcConf(storedRuxitProcResponse.Revision)
	if err != nil {
		return nil, lastRevision, err
	}
	if latestRuxitProcResponse != nil {
		return latestRuxitProcResponse, lastRevision, nil
	}
	return storedRuxitProcResponse, lastRevision, nil
}

func (r *OneAgentProvisioner) readRuxitCache(tenantUUID string) (*dtclient.RuxitProcResponse, error) {
	var ruxitConf dtclient.RuxitProcResponse
	ruxitProcCache, err := r.fs.Open(r.path.AgentRuxitProcResponseCache(tenantUUID))
	if err != nil {
		return nil, err
	}
	jsonBytes, err := ioutil.ReadAll(ruxitProcCache)
	ruxitProcCache.Close()
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(jsonBytes, &ruxitConf); err != nil {
		return nil, err
	}

	return &ruxitConf, nil
}

func (r *OneAgentProvisioner) writeRuxitCache(tenantUUID string, ruxitResponse *dtclient.RuxitProcResponse) error {
	ruxitProcCache, err := r.fs.OpenFile(r.path.AgentRuxitProcResponseCache(tenantUUID), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	jsonBytes, err := json.Marshal(ruxitResponse)
	if err != nil {
		ruxitProcCache.Close()
		return err
	}
	_, err = ruxitProcCache.Write(jsonBytes)
	ruxitProcCache.Close()
	return err
}

func (installAgentCfg *installAgentConfig) updateRuxitConf(version, tenantUUID string, ruxitResponse *dtclient.RuxitProcResponse) error {
	if ruxitResponse != nil {
		ruxitConf := ruxitResponse.ToMap()
		installAgentCfg.logger.Info("updating ruxitagentproc.conf", "agentVersion", version, "tenantUUID", tenantUUID)
		usedRuxitConfPath := installAgentCfg.path.AgentRuxitConfForVersion(tenantUUID, version)
		sourceRuxitConfPath := installAgentCfg.path.SourceAgentRuxitConfForVersion(tenantUUID, version)
		if err := installAgentCfg.checkRuxitConfCopy(sourceRuxitConfPath, usedRuxitConfPath); err != nil {
			return err
		}
		return conf.UpdateConfFile(installAgentCfg.fs, sourceRuxitConfPath, usedRuxitConfPath, ruxitConf)
	}
	installAgentCfg.logger.Info("no changes to ruxitagentproc.conf, skipping update")
	return nil
}

// checkRuxitConfCopy checks if we already made a copy of the original ruxitagentproc.conf file.
// After the initial install of a version we copy the ruxitagentproc.conf to _ruxitagentproc.conf and we use the _ruxitagentproc.conf + the api response to re-create the ruxitagentproc.conf
// so its easier to update
func (installAgentCfg *installAgentConfig) checkRuxitConfCopy(sourcePath, destPath string) error {
	if _, err := installAgentCfg.fs.Open(sourcePath); os.IsNotExist(err) {
		fileInfo, err := installAgentCfg.fs.Stat(destPath)
		if err != nil {
			return err
		}

		sourceRuxitConfFile, err := installAgentCfg.fs.OpenFile(sourcePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fileInfo.Mode())
		if err != nil {
			return err
		}

		usedRuxitConfFile, err := installAgentCfg.fs.Open(destPath)
		if err != nil {
			return err
		}
		_, err = io.Copy(sourceRuxitConfFile, usedRuxitConfFile)
		if err != nil {
			sourceRuxitConfFile.Close()
			usedRuxitConfFile.Close()
			return err
		}
		return usedRuxitConfFile.Close()
	}
	return nil
}

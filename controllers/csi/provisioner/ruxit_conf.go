package csiprovisioner

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"os"

	ruxit "github.com/Dynatrace/dynatrace-operator/conf"
	"github.com/Dynatrace/dynatrace-operator/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
)

// getRuxitProcResponse gets the latest `RuxitProcResponse`, it can come from the tenant if we don't have the latest revision saved locally,
// otherwise we use the locally cached response
func (r *OneAgentProvisioner) getRuxitProcResponse(ruxitRevision *metadata.RuxitRevision, dtc dtclient.Client) (*dtclient.RuxitProcResponse, error) {
	var latestRevision uint
	if ruxitRevision.LatestRevision != 0 {
		latestRevision = ruxitRevision.LatestRevision
	}

	latestRuxitProcResponse, err := dtc.GetRuxitProcConf(latestRevision)
	if err != nil {
		return nil, err
	}

	storedRuxitProcResponse, err := r.readRuxitCache(ruxitRevision)
	if err != nil && os.IsNotExist(err) && latestRuxitProcResponse == nil {
		latestRuxitProcResponse, err = dtc.GetRuxitProcConf(0)
		if err != nil {
			return nil, err
		}
	} else if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	if latestRuxitProcResponse != nil {
		return latestRuxitProcResponse, nil
	}
	return storedRuxitProcResponse, nil
}

func (r *OneAgentProvisioner) readRuxitCache(ruxitRevision *metadata.RuxitRevision) (*dtclient.RuxitProcResponse, error) {
	var ruxitConf dtclient.RuxitProcResponse
	ruxitProcCache, err := r.fs.Open(r.path.AgentRuxitProcResponseCache(ruxitRevision.TenantUUID))
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

func (r *OneAgentProvisioner) writeRuxitCache(ruxitRevision *metadata.RuxitRevision, ruxitResponse *dtclient.RuxitProcResponse) error {
	ruxitProcCache, err := r.fs.OpenFile(r.path.AgentRuxitProcResponseCache(ruxitRevision.TenantUUID), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
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

func (r *OneAgentProvisioner) createOrUpdateRuxitRevision(tenantUUID string, ruxitRevision *metadata.RuxitRevision, ruxitResponse *dtclient.RuxitProcResponse) error {
	if ruxitRevision.LatestRevision == 0 && ruxitResponse != nil {
		log.Info("inserting ruxit revision into db", "tenantUUID", tenantUUID, "revision", ruxitResponse.Revision)
		return r.db.InsertRuxitRevision(metadata.NewRuxitRevision(tenantUUID, ruxitResponse.Revision))
	} else if ruxitResponse != nil && ruxitResponse.Revision != ruxitRevision.LatestRevision {
		log.Info("updating ruxit revision in db", "tenantUUID", tenantUUID, "old-revision", ruxitRevision.LatestRevision, "new-revision", ruxitResponse.Revision)
		ruxitRevision.LatestRevision = ruxitResponse.Revision
		return r.db.UpdateRuxitRevision(ruxitRevision)
	}
	return nil
}

func (installAgentCfg *installAgentConfig) updateRuxitConf(version, tenantUUID string, ruxitResponse *dtclient.RuxitProcResponse) error {
	if ruxitResponse != nil {
		conf := ruxitResponse.ToMap()
		installAgentCfg.logger.Info("updating ruxitagentproc.conf", "agentVersion", version, "tenantUUID", tenantUUID)
		usedRuxitConfPath := installAgentCfg.path.AgentRuxitConfForVersion(tenantUUID, version)
		sourceRuxitConfPath := installAgentCfg.path.SourceAgentRuxitConfForVersion(tenantUUID, version)
		if err := installAgentCfg.checkRuxitConfCopy(sourceRuxitConfPath, usedRuxitConfPath); err != nil {
			return err
		}
		return ruxit.UpdateConfFile(installAgentCfg.fs, sourceRuxitConfPath, usedRuxitConfPath, conf)
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

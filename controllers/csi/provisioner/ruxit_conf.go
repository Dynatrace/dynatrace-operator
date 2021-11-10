package csiprovisioner

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
)

type ruxitConfPatch struct {
	updateMap dtclient.RuxitProcMap
}

var confSectionRegexp, _ = regexp.Compile(`\[(.*)\]`)

func (r *OneAgentProvisioner) getRuxitProcConf(ruxitRevission *metadata.RuxitRevision, dtc dtclient.Client) (*dtclient.RuxitProcResponse, *ruxitConfPatch, error) {
	var latestRevission uint
	if ruxitRevission.LatestRevission != 0 {
		latestRevission = ruxitRevission.LatestRevission
	}

	latestRuxitConfResponse, err := dtc.GetRuxitProcConf(latestRevission)
	if err != nil {
		return nil, nil, err
	}

	storedRuxitConfResponse, err := r.readRuxitCache(ruxitRevission)
	if err != nil && os.IsNotExist(err) && latestRuxitConfResponse == nil {
		latestRuxitConfResponse, err = dtc.GetRuxitProcConf(0)
		if err != nil {
			return nil, nil, err
		}
	} else if err != nil && !os.IsNotExist(err) {
		return nil, nil, err
	}

	if storedRuxitConfResponse == nil {
		storedRuxitConfResponse = latestRuxitConfResponse
	} else if latestRuxitConfResponse == nil {
		latestRuxitConfResponse = storedRuxitConfResponse
	}

	confPatch := createRuxitConfPatch(latestRuxitConfResponse, storedRuxitConfResponse)

	return latestRuxitConfResponse, confPatch, nil
}

func createRuxitConfPatch(latestRuxitConfResponse, storedRuxitConfResponse *dtclient.RuxitProcResponse) *ruxitConfPatch {
	confPatch := ruxitConfPatch{}
	if latestRuxitConfResponse != nil {
		confPatch = ruxitConfPatch{
			updateMap: latestRuxitConfResponse.ToMap(),
		}
	} else {
		confPatch = ruxitConfPatch{
			updateMap: storedRuxitConfResponse.ToMap(),
		}
	}
	return &confPatch
}

func (r *OneAgentProvisioner) readRuxitCache(ruxitRevission *metadata.RuxitRevision) (*dtclient.RuxitProcResponse, error) {
	var ruxitConf dtclient.RuxitProcResponse
	ruxitConfCache, err := r.fs.Open(r.path.AgentRuxitRevision(ruxitRevission.TenantUUID))
	if err != nil {
		return nil, err
	}
	jsonBytes, err := ioutil.ReadAll(ruxitConfCache)
	ruxitConfCache.Close()
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(jsonBytes, &ruxitConf); err != nil {
		return nil, err
	}

	return &ruxitConf, nil
}

func (r *OneAgentProvisioner) writeRuxitCache(ruxitRevission *metadata.RuxitRevision, ruxitConf *dtclient.RuxitProcResponse) error {
	ruxitConfFile, err := r.fs.OpenFile(r.path.AgentRuxitRevision(ruxitRevission.TenantUUID), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	jsonBytes, err := json.Marshal(ruxitConf)
	if err != nil {
		ruxitConfFile.Close()
		return err
	}
	_, err = ruxitConfFile.Write(jsonBytes)
	ruxitConfFile.Close()
	return err
}

func (r *OneAgentProvisioner) createOrUpdateRuxitRevision(tenantUUID string, ruxitRevision *metadata.RuxitRevision, ruxitConf *dtclient.RuxitProcResponse) error {
	if ruxitRevision.LatestRevission == 0 && ruxitConf != nil {
		log.Info("inserting ruxit revission into db", "tenantUUID", tenantUUID, "revission", ruxitConf.Revision)
		return r.db.InsertRuxitRevission(metadata.NewRuxitRevission(tenantUUID, ruxitConf.Revision))
	} else if ruxitConf != nil && ruxitConf.Revision != ruxitRevision.LatestRevission {
		log.Info("updating ruxit revission in db", "tenantUUID", tenantUUID, "old-revission", ruxitRevision.LatestRevission, "new-revission", ruxitConf.Revision)
		ruxitRevision.LatestRevission = ruxitConf.Revision
		return r.db.UpdateRuxitRevission(ruxitRevision)
	}
	return nil
}

func (installAgentCfg *installAgentConfig) updateRuxitConf(version, tenantUUID string, confPatch *ruxitConfPatch) error {
	if confPatch != nil {
		installAgentCfg.logger.Info("updating ruxitagentproc.conf", "agentVersion", version, "tenantUUID", tenantUUID)
		confContent, err := installAgentCfg.mergeRuxitConf(version, tenantUUID, confPatch)
		if err != nil {
			return err
		}

		// for sections not in the conf file found in the zip
		confContent = append(confContent, addLeftovers(confPatch.updateMap)...)

		return installAgentCfg.storeRuxitConf(version, tenantUUID, confContent)
	}
	installAgentCfg.logger.Info("no changes to ruxitagentproc.conf, skipping update")
	return nil
}

func (installAgentCfg *installAgentConfig) mergeRuxitConf(version, tenantUUID string, confPatch *ruxitConfPatch) ([]string, error) {
	usedRuxitConfPath := installAgentCfg.path.AgentRuxitConfForVersion(tenantUUID, version)
	sourceRuxitConfPath := installAgentCfg.path.SourceAgentRuxitConfForVersion(tenantUUID, version)
	sourceRuxitConfFile, err := installAgentCfg.fs.Open(sourceRuxitConfPath)

	// After the initial install of a version we copy the ruxitagentproc.conf to _ruxitagentproc.conf and we use the _ruxitagentproc.conf + the api response to re-create the ruxitagentproc.conf
	// so its easier to update
	if os.IsNotExist(err) {
		// TODO: Migrate into a function
		fileInfo, err := installAgentCfg.fs.Stat(usedRuxitConfPath)
		if err != nil {
			return nil, err
		}

		sourceRuxitConfFile, err = installAgentCfg.fs.OpenFile(sourceRuxitConfPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fileInfo.Mode())
		if err != nil {
			return nil, err
		}

		usedRuxitConfFile, err := installAgentCfg.fs.Open(usedRuxitConfPath)
		if err != nil {
			return nil, err
		}
		_, err = io.Copy(sourceRuxitConfFile, usedRuxitConfFile)
		if err != nil {
			return nil, err
		}
	}
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(sourceRuxitConfFile)
	currentSection := ""
	finalLines := []string{}
	for scanner.Scan() {
		line := scanner.Text()
		if header := confSectionHeader(line); header != "" {
			finalLines = append(finalLines, addLeftoversForSection(currentSection, confPatch.updateMap)...)
			currentSection = header
			finalLines = append(finalLines, line)
			installAgentCfg.logger.Info("ruxitagentproc.conf updating", "section", currentSection)
		} else if strings.HasPrefix(line, "#") {
			finalLines = append(finalLines, line)
		} else {
			finalLines = append(finalLines, mergeLine(line, currentSection, confPatch))
		}
	}
	if err := scanner.Err(); err != nil {
		sourceRuxitConfFile.Close()
		return nil, err
	}

	// the last section's leftover cleanup never runs in the for loop
	finalLines = append(finalLines, addLeftoversForSection(currentSection, confPatch.updateMap)...)

	return finalLines, sourceRuxitConfFile.Close()
}

func (installAgentCfg *installAgentConfig) storeRuxitConf(version, tenantUUID string, content []string) error {
	ruxitConfPath := installAgentCfg.path.AgentRuxitConfForVersion(tenantUUID, version)
	fileInfo, err := installAgentCfg.fs.Stat(ruxitConfPath)
	if err != nil {
		return err
	}
	ruxitConfFile, err := installAgentCfg.fs.OpenFile(ruxitConfPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fileInfo.Mode())
	if err != nil {
		return err
	}
	for _, line := range content {
		_, _ = ruxitConfFile.WriteString(line + "\n")
		if err != nil {
			ruxitConfFile.Close()
			return err
		}
	}
	return ruxitConfFile.Close()
}

func confSectionHeader(confLine string) string {
	if matches := confSectionRegexp.FindStringSubmatch(confLine); len(matches) != 0 {
		return matches[1]
	}
	return ""
}

func addLeftovers(ruxitConf dtclient.RuxitProcMap) []string {
	lines := []string{}
	for section, props := range ruxitConf {
		lines = append(lines, fmt.Sprintf("[%s]", section)) // TODO: should add logs here
		for key, value := range props {
			lines = append(lines, fmt.Sprintf("%s %s", key, value))
		}
	}
	return lines
}

func addLeftoversForSection(currentSection string, ruxitConf dtclient.RuxitProcMap) []string {
	lines := []string{}
	if currentSection != "" {
		section, ok := ruxitConf[currentSection]
		if ok {
			for key, value := range section {
				lines = append(lines, fmt.Sprintf("%s %s", key, value))
			}
			delete(ruxitConf, currentSection)
		}
	}
	return lines
}

func mergeLine(line, currentSection string, confPatch *ruxitConfPatch) string {
	splitLine := strings.Split(line, " ")
	key := splitLine[0]

	props, ok := confPatch.updateMap[currentSection]
	if !ok {
		return line
	}
	newValue, ok := props[key]
	if !ok {
		return line
	}
	delete(confPatch.updateMap[currentSection], key)
	return fmt.Sprintf("%s %s", key, newValue)
}

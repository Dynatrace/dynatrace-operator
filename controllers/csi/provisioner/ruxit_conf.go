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
	"github.com/spf13/afero"
)

var confSectionRegexp, _ = regexp.Compile(`\[(.*)\]`)

func (r *OneAgentProvisioner) getRuxitProcResponse(ruxitRevission *metadata.RuxitRevision, dtc dtclient.Client) (*dtclient.RuxitProcResponse, error) {
	var latestRevission uint
	if ruxitRevission.LatestRevission != 0 {
		latestRevission = ruxitRevission.LatestRevission
	}

	latestRuxitProcResponse, err := dtc.GetRuxitProcConf(latestRevission)
	if err != nil {
		return nil, err
	}

	storedRuxitProcResponse, err := r.readRuxitCache(ruxitRevission)
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

func (r *OneAgentProvisioner) writeRuxitCache(ruxitRevission *metadata.RuxitRevision, ruxitResponse *dtclient.RuxitProcResponse) error {
	ruxitConfFile, err := r.fs.OpenFile(r.path.AgentRuxitRevision(ruxitRevission.TenantUUID), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	jsonBytes, err := json.Marshal(ruxitResponse)
	if err != nil {
		ruxitConfFile.Close()
		return err
	}
	_, err = ruxitConfFile.Write(jsonBytes)
	ruxitConfFile.Close()
	return err
}

func (r *OneAgentProvisioner) createOrUpdateRuxitRevision(tenantUUID string, ruxitRevision *metadata.RuxitRevision, ruxitResponse *dtclient.RuxitProcResponse) error {
	if ruxitRevision.LatestRevission == 0 && ruxitResponse != nil {
		log.Info("inserting ruxit revission into db", "tenantUUID", tenantUUID, "revission", ruxitResponse.Revision)
		return r.db.InsertRuxitRevission(metadata.NewRuxitRevission(tenantUUID, ruxitResponse.Revision))
	} else if ruxitResponse != nil && ruxitResponse.Revision != ruxitRevision.LatestRevission {
		log.Info("updating ruxit revission in db", "tenantUUID", tenantUUID, "old-revission", ruxitRevision.LatestRevission, "new-revission", ruxitResponse.Revision)
		ruxitRevision.LatestRevission = ruxitResponse.Revision
		return r.db.UpdateRuxitRevission(ruxitRevision)
	}
	return nil
}

func (installAgentCfg *installAgentConfig) updateRuxitConf(version, tenantUUID string, ruxitResponse *dtclient.RuxitProcResponse) error {
	if ruxitResponse != nil {
		procMap := ruxitResponse.ToMap()
		installAgentCfg.logger.Info("updating ruxitagentproc.conf", "agentVersion", version, "tenantUUID", tenantUUID)
		return installAgentCfg.mergeRuxitConf(version, tenantUUID, procMap)
	}
	installAgentCfg.logger.Info("no changes to ruxitagentproc.conf, skipping update")
	return nil
}

func (installAgentCfg *installAgentConfig) mergeRuxitConf(version, tenantUUID string, procMap dtclient.RuxitProcMap) error {
	usedRuxitConfPath := installAgentCfg.path.AgentRuxitConfForVersion(tenantUUID, version)
	sourceRuxitConfPath := installAgentCfg.path.SourceAgentRuxitConfForVersion(tenantUUID, version)
	sourceRuxitConfFile, err := installAgentCfg.fs.Open(sourceRuxitConfPath)

	// After the initial install of a version we copy the ruxitagentproc.conf to _ruxitagentproc.conf and we use the _ruxitagentproc.conf + the api response to re-create the ruxitagentproc.conf
	// so its easier to update
	if os.IsNotExist(err) {
		// TODO: Migrate into a function
		fileInfo, err := installAgentCfg.fs.Stat(usedRuxitConfPath)
		if err != nil {
			return err
		}

		sourceRuxitConfFile, err = installAgentCfg.fs.OpenFile(sourceRuxitConfPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fileInfo.Mode())
		if err != nil {
			return err
		}

		usedRuxitConfFile, err := installAgentCfg.fs.Open(usedRuxitConfPath)
		if err != nil {
			return err
		}
		_, err = io.Copy(sourceRuxitConfFile, usedRuxitConfFile)
		if err != nil {
			sourceRuxitConfFile.Close()
			usedRuxitConfFile.Close()
			return err
		}
		usedRuxitConfFile.Close()
	}
	sourceRuxitConfFile.Close()
	if err != nil {
		return err
	}
	return updateConfFile(installAgentCfg.fs, sourceRuxitConfPath, usedRuxitConfPath, procMap)
}

// TODO: move into other/separate package
func updateConfFile(fs afero.Fs, sourcePath, destPath string, procMap dtclient.RuxitProcMap) error {
	sourceFile, err := fs.Open(sourcePath)
	if err != nil {
		return err
	}
	scanner := bufio.NewScanner(sourceFile)
	currentSection := ""
	content := []string{}
	for scanner.Scan() {
		line := scanner.Text()
		if header := confSectionHeader(line); header != "" {
			content = append(content, addLeftoversForSection(currentSection, procMap)...)
			currentSection = header
			content = append(content, line)
		} else if strings.HasPrefix(line, "#") {
			content = append(content, line)
		} else {
			content = append(content, mergeLine(line, currentSection, procMap))
		}
	}
	if err := scanner.Err(); err != nil {
		sourceFile.Close()
		return err
	}

	// the last section's leftover cleanup never runs in the for loop
	content = append(content, addLeftoversForSection(currentSection, procMap)...)

	// for sections not in the conf file found in the zip
	content = append(content, addLeftovers(procMap)...)

	if err = sourceFile.Close(); err != nil {
		return err
	}

	return storeConfFile(fs, destPath, content)
}

func storeConfFile(fs afero.Fs, destPath string, content []string) error {
	fileInfo, err := fs.Stat(destPath)
	if err != nil {
		return err
	}
	ruxitConfFile, err := fs.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fileInfo.Mode())
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

func addLeftovers(procMap dtclient.RuxitProcMap) []string {
	lines := []string{}
	for section, props := range procMap {
		lines = append(lines, fmt.Sprintf("[%s]", section)) // TODO: should add logs here
		for key, value := range props {
			lines = append(lines, fmt.Sprintf("%s %s", key, value))
		}
	}
	return lines
}

func addLeftoversForSection(currentSection string, procMap dtclient.RuxitProcMap) []string {
	lines := []string{}
	if currentSection != "" {
		section, ok := procMap[currentSection]
		if ok {
			for key, value := range section {
				lines = append(lines, fmt.Sprintf("%s %s", key, value))
			}
			delete(procMap, currentSection)
		}
	}
	return lines
}

func mergeLine(line, currentSection string, procMap dtclient.RuxitProcMap) string {
	splitLine := strings.Split(line, " ")
	key := splitLine[0]

	props, ok := procMap[currentSection]
	if !ok {
		return line
	}
	newValue, ok := props[key]
	if !ok {
		return line
	}
	delete(procMap[currentSection], key)
	return fmt.Sprintf("%s %s", key, newValue)
}

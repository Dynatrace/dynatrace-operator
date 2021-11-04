package csiprovisioner

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

var confSectionRegexp, _ = regexp.Compile(`\[(.*)\]`)

func (installAgentCfg *installAgentConfig) updateRuxitConf(version, tenantUUID string, ruxitConf map[string]map[string]string) error {
	if ruxitConf != nil {
		installAgentCfg.logger.Info("updating ruxitagentproc.conf")
		confContent, err := installAgentCfg.mergeRuxitConf(version, tenantUUID, ruxitConf)
		if err != nil {
			return err
		}

		// for sections not in the conf file found in the zip
		confContent = append(confContent, addLeftovers(ruxitConf)...)

		return installAgentCfg.storeRuxitConf(version, tenantUUID, confContent)
	}
	installAgentCfg.logger.Info("no changes to ruxitagentproc.conf, skipping update")
	return nil
}

func (installAgentCfg *installAgentConfig) mergeRuxitConf(version, tenantUUID string, ruxitConf map[string]map[string]string) ([]string, error) {
	ruxitConfPath := installAgentCfg.path.AgentRuxitConfForVersion(tenantUUID, version)
	ruxitConfFile, err := installAgentCfg.fs.Open(ruxitConfPath)
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(ruxitConfFile)
	currentSection := ""
	finalLines := []string{}
	for scanner.Scan() {
		line := scanner.Text()
		if header := confSectionHeader(line); header != "" {
			finalLines = append(finalLines, addLeftoversForSection(currentSection, ruxitConf)...)
			currentSection = header
			finalLines = append(finalLines, line)
			installAgentCfg.logger.Info("ruxitagentproc.conf updating", "section", currentSection)
		} else if strings.HasPrefix(line, "#") {
			finalLines = append(finalLines, line)
		} else {
			finalLines = append(finalLines, mergeLine(line, currentSection, ruxitConf))
		}
	}
	if err := scanner.Err(); err != nil {
		ruxitConfFile.Close()
		return nil, err
	}

	// the last section's leftover cleanup never runs in the for loop
	finalLines = append(finalLines, addLeftoversForSection(currentSection, ruxitConf)...)

	return finalLines, ruxitConfFile.Close()
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

func addLeftovers(ruxitConf map[string]map[string]string) []string {
	lines := []string{}
	for section, props := range ruxitConf {
		lines = append(lines, fmt.Sprintf("[%s]", section)) // TODO: should add logs here
		for key, value := range props {
			lines = append(lines, fmt.Sprintf("%s %s", key, value))
		}
	}
	return lines
}

func addLeftoversForSection(currentSection string, ruxitConf map[string]map[string]string) []string {
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

func mergeLine(line, currentSection string, ruxitConf map[string]map[string]string) string {
	splitLine := strings.Split(line, " ")
	key := splitLine[0]
	props, ok := ruxitConf[currentSection]
	if !ok {
		return line
	}
	newValue, ok := props[key]
	if !ok {
		return line
	}
	delete(ruxitConf[currentSection], key)
	return fmt.Sprintf("%s %s", key, newValue)
}

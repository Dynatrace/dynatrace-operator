package types

const generalSectionName = "general"

type ProcessModuleConfig struct {
	Revision   uint                    `json:"revision"`
	Properties []ProcessModuleProperty `json:"properties"`
}

type ProcessModuleProperty struct {
	Section string `json:"section"`
	Key     string `json:"key"`
	Value   string `json:"value"`
}

// ConfMap is the representation of a config file with sections that are divided by headers (map[header] == section)
// each section consists of key value pairs.
type ConfMap map[string]map[string]string

func (pmc *ProcessModuleConfig) Add(newProperty ProcessModuleProperty) *ProcessModuleConfig {
	if pmc == nil {
		pmc = &ProcessModuleConfig{}
	}

	var newProps []ProcessModuleProperty
	hasPropertyGroup := false
	for _, currentProperty := range pmc.Properties {
		if currentProperty.Key != newProperty.Key {
			newProps = append(newProps, currentProperty)
		} else {
			hasPropertyGroup = true
			if newProperty.Value == "" {
				continue
			} else if newProperty.Value == currentProperty.Value {
				newProps = append(newProps, currentProperty)
			} else {
				newProps = append(pmc.Properties, currentProperty)
			}
		}
	}
	if !hasPropertyGroup && newProperty.Value != "" {
		newProps = append(pmc.Properties, newProperty)
	}
	pmc.Properties = newProps
	return pmc
}

func (pmc *ProcessModuleConfig) AddConnectionInfo(connectionInfo ConnectionInfo) *ProcessModuleConfig {
	tenant := ProcessModuleProperty{
		Section: generalSectionName,
		Key:     "tenant",
		Value:   connectionInfo.TenantUUID,
	}
	pmc.Add(tenant)

	token := ProcessModuleProperty{
		Section: generalSectionName,
		Key:     "tenantToken",
		Value:   connectionInfo.TenantToken,
	}
	pmc.Add(token)

	endpoints := ProcessModuleProperty{
		Section: generalSectionName,
		Key:     "serverAddress",
		Value:   "{" + connectionInfo.FormattedCommunicationEndpoints + "}",
	}
	pmc.Add(endpoints)

	return pmc
}

func (pmc *ProcessModuleConfig) AddHostGroup(hostGroup string) *ProcessModuleConfig {
	property := ProcessModuleProperty{Section: generalSectionName, Key: "hostGroup", Value: hostGroup}
	return pmc.Add(property)
}

func (pmc ProcessModuleConfig) ToMap() ConfMap {
	sections := map[string]map[string]string{}
	for _, prop := range pmc.Properties {
		section := sections[prop.Section]
		if section == nil {
			section = map[string]string{}
		}
		section[prop.Key] = prop.Value
		sections[prop.Section] = section
	}
	return sections
}

func (pmc ProcessModuleConfig) IsEmpty() bool {
	return len(pmc.Properties) == 0
}

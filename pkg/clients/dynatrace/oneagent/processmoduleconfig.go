package oneagent

import (
	"context"
	"encoding/json"
	"slices"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/pkg/errors"
)

const (
	generalSectionName      = "general"
	hostGroupParamName      = "hostgroup"
	processModuleConfigPath = "/v1/deployment/installer/agent/processmoduleconfig"
)

type ProcessModuleConfig struct {
	Properties []ProcessModuleProperty `json:"properties"`
	Revision   uint                    `json:"revision"`
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
	pmc.fixBrokenCache()

	for index, cachedProperty := range pmc.Properties {
		if cachedProperty.Key == newProperty.Key {
			if newProperty.Value == "" {
				pmc.removeProperty(index)
			} else {
				pmc.updateProperty(index, newProperty)
			}

			return pmc
		}
	}

	if newProperty.Value != "" {
		pmc.addProperty(newProperty)
	}

	return pmc
}

// fixBrokenCache fixes a cache that might have been broken by previous versions
// Older operator versions handled the cache wrong and multiplied properties on an update
// instead of updating it.
// The fixed algorithm in Add cannot handle this broken cache without this function
// It adds every property first to a map using the property's key, to make them distinct
// Then collects the now distinct properties and updates the cache
func (pmc *ProcessModuleConfig) fixBrokenCache() {
	properties := make([]ProcessModuleProperty, 0, len(pmc.Properties))
	propertyMap := make(map[string]ProcessModuleProperty)

	for _, property := range pmc.Properties {
		propertyMap[property.Key] = property
	}

	for _, value := range propertyMap {
		properties = append(properties, value)
	}

	pmc.Properties = properties
}

func (pmc *ProcessModuleConfig) addProperty(newProperty ProcessModuleProperty) {
	pmc.Properties = append(pmc.Properties, newProperty)
}

func (pmc *ProcessModuleConfig) updateProperty(index int, newProperty ProcessModuleProperty) {
	pmc.Properties[index].Section = newProperty.Section
	pmc.Properties[index].Value = newProperty.Value
}

func (pmc *ProcessModuleConfig) removeProperty(index int) {
	pmc.Properties = append(pmc.Properties[0:index], pmc.Properties[index+1:]...)
}

func (pmc *ProcessModuleConfig) AddConnectionInfo(oneAgentConnectionInfo communication.ConnectionInfo, tenantToken string) *ProcessModuleConfig {
	tenant := ProcessModuleProperty{
		Section: generalSectionName,
		Key:     "tenant",
		Value:   oneAgentConnectionInfo.TenantUUID,
	}
	pmc.Add(tenant)

	token := ProcessModuleProperty{
		Section: generalSectionName,
		Key:     "tenantToken",
		Value:   tenantToken,
	}
	pmc.Add(token)

	endpoints := ProcessModuleProperty{
		Section: generalSectionName,
		Key:     "serverAddress",
		Value:   "{" + oneAgentConnectionInfo.Endpoints + "}",
	}
	pmc.Add(endpoints)

	return pmc
}

func (pmc *ProcessModuleConfig) AddHostGroup(hostGroup string) *ProcessModuleConfig {
	property := ProcessModuleProperty{Section: generalSectionName, Key: "hostGroup", Value: hostGroup}

	return pmc.Add(property)
}

func (pmc *ProcessModuleConfig) AddProxy(proxy string) *ProcessModuleConfig {
	property := ProcessModuleProperty{Section: generalSectionName, Key: "proxy", Value: proxy}

	return pmc.Add(property)
}

func (pmc *ProcessModuleConfig) AddNoProxy(noProxy string) *ProcessModuleConfig {
	property := ProcessModuleProperty{Section: generalSectionName, Key: "noProxy", Value: noProxy}

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

func (pmc *ProcessModuleConfig) SortPropertiesByKey() {
	slices.SortFunc(pmc.Properties, func(a, b ProcessModuleProperty) int {
		return strings.Compare(a.Key, b.Key)
	})
}

func (pmc ProcessModuleConfig) IsEmpty() bool {
	return len(pmc.Properties) == 0
}

func (c *Client) GetProcessModuleConfig(ctx context.Context) (*ProcessModuleConfig, error) {
	var resp ProcessModuleConfig

	params := map[string]string{
		"sections": "general,agentType",
	}

	if c.hostGroup != "" {
		params[hostGroupParamName] = c.hostGroup
	}

	err := c.apiClient.GET(ctx, processModuleConfigPath).
		WithPaasToken().
		WithQueryParams(params).
		Execute(&resp)
	if err != nil {
		return &ProcessModuleConfig{}, errors.WithMessage(err, "error while requesting process module config")
	}

	if len(resp.Properties) == 0 {
		return &ProcessModuleConfig{}, errors.New("no properties available")
	}

	return &resp, nil
}

func NewProcessModuleConfig(ctx context.Context, response []byte) (*ProcessModuleConfig, error) {
	log := logd.FromContext(ctx)

	resp := ProcessModuleConfig{}

	err := json.Unmarshal(response, &resp)
	if err != nil {
		log.Error(err, "error unmarshalling processmoduleconfig response", "response", string(response))

		return nil, err
	}

	if len(resp.Properties) == 0 {
		return nil, errors.New("no properties available")
	}

	return &resp, nil
}

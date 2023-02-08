package dtclient

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/pkg/errors"
)

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

func (pmc *ProcessModuleConfig) AddConnectionInfo(connectionInfo OneAgentConnectionInfo) *ProcessModuleConfig {
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
		Value:   "{" + connectionInfo.Endpoints + "}",
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

func (dtc *dynatraceClient) GetProcessModuleConfig(prevRevision uint) (*ProcessModuleConfig, error) {
	req, err := dtc.createProcessModuleConfigRequest(prevRevision)
	if err != nil {
		return nil, err
	}

	resp, err := dtc.httpClient.Do(req)

	if dtc.checkProcessModuleConfigRequestStatus(resp) {
		return &ProcessModuleConfig{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error while requesting process module config: %w", err)
	}
	defer CloseBodyAfterRequest(resp)

	responseData, err := dtc.getServerResponseData(resp)
	if err != nil {
		return nil, err
	}

	return dtc.readResponseForProcessModuleConfig(responseData)
}

func (dtc *dynatraceClient) createProcessModuleConfigRequest(prevRevision uint) (*http.Request, error) {
	req, err := http.NewRequest(http.MethodGet, dtc.getProcessModuleConfigUrl(), nil)
	if err != nil {
		return nil, fmt.Errorf("error initializing http request: %w", err)
	}
	query := req.URL.Query()
	query.Add("revision", strconv.FormatUint(uint64(prevRevision), 10))
	req.URL.RawQuery = query.Encode()
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Api-Token %s", dtc.paasToken))

	return req, nil
}

// The endpoint used here is new therefore some tenants may not have it so we need to
// handle it gracefully, by checking the status code of the request.
// we also handle when there were no changes
func (dtc *dynatraceClient) checkProcessModuleConfigRequestStatus(resp *http.Response) bool {
	if resp == nil {
		log.Info("problems checking response for processmoduleconfig endpoint")
		return true
	}

	if resp.StatusCode == http.StatusNotModified {
		return true
	}

	if resp.StatusCode == http.StatusNotFound {
		log.Info("endpoint for ruxitagentproc.conf is not available on the cluster.")
		return true
	}

	return false
}

func (dtc *dynatraceClient) readResponseForProcessModuleConfig(response []byte) (*ProcessModuleConfig, error) {
	resp := ProcessModuleConfig{}
	err := json.Unmarshal(response, &resp)
	if err != nil {
		log.Error(err, "error unmarshalling processmoduleconfig response", "response", string(response))
		return nil, err
	}

	if len(resp.Properties) == 0 {
		return nil, errors.New("no communication hosts available")
	}

	return &resp, nil
}

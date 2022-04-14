package dtclient

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/Dynatrace/dynatrace-operator/src/processmoduleconfig"
	"github.com/pkg/errors"
)

type ProcessModuleConfig struct {
	Revision   uint                    `json:"revision"`
	Properties []ProcessModuleProperty `json:"properties"`
}

type ProcessModuleProperty struct {
	Section string `json:"section"`
	Key     string `json:"key"`
	Value   string `json:"value"`
}

func (pmc *ProcessModuleConfig) Add(newProperty ProcessModuleProperty) *ProcessModuleConfig {
	if pmc == nil {
		return &ProcessModuleConfig{}
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
		Section: "general",
		Key:     "tenant",
		Value:   connectionInfo.TenantUUID,
	}
	pmc.Add(tenant)

	token := ProcessModuleProperty{
		Section: "general",
		Key:     "tenantToken",
		Value:   connectionInfo.TenantToken,
	}
	pmc.Add(token)

	endpoints := ProcessModuleProperty{
		Section: "general",
		Key:     "serverAddress",
		Value:   "{" + connectionInfo.FormattedCommunicationEndpoints + "}",
	}
	pmc.Add(endpoints)

	return pmc
}

func (pmc *ProcessModuleConfig) AddHostGroup(hostGroup string) *ProcessModuleConfig {
	property := ProcessModuleProperty{Section: "general", Key: "hostGroup", Value: hostGroup}
	return pmc.Add(property)
}

func (pmc ProcessModuleConfig) ToMap() processmoduleconfig.ConfMap {
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
		return nil, fmt.Errorf("error while requesting process module config: %v", err)
	}
	defer func() {
		//Swallow error, nothing has to be done at this point
		_ = resp.Body.Close()
	}()

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

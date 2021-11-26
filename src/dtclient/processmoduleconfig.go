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

func (dtc *dynatraceClient) GetProcessModuleConfig(prevRevision uint) (*ProcessModuleConfig, error) {
	req, err := dtc.createProcessModuleConfigRequest(prevRevision)
	if err != nil {
		return nil, err
	}

	resp, err := dtc.httpClient.Do(req)

	if dtc.specialProcessModuleConfigRequestStatus(resp) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error making get request to dynatrace api: %w", err)
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
	processModuleConfigURL := fmt.Sprintf("%s/v1/deployment/installer/agent/processmoduleconfig", dtc.url)

	req, err := http.NewRequest(http.MethodGet, processModuleConfigURL, nil)
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

func (dtc *dynatraceClient) specialProcessModuleConfigRequestStatus(resp *http.Response) bool {
	if resp.StatusCode == http.StatusNotModified {
		return true
	}

	if resp.StatusCode == http.StatusNotFound {
		dtc.logger.Info("endpoint for ruxitagentproc.conf is not available on the cluster.")
		return true
	}

	return false
}

func (dtc *dynatraceClient) readResponseForProcessModuleConfig(response []byte) (*ProcessModuleConfig, error) {
	resp := ProcessModuleConfig{}
	err := json.Unmarshal(response, &resp)
	if err != nil {
		dtc.logger.Error(err, "error unmarshalling json response")
		return nil, err
	}

	if len(resp.Properties) == 0 {
		return nil, errors.New("no communication hosts available")
	}

	return &resp, nil
}

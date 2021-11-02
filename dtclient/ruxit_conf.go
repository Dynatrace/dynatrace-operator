package dtclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
)

const notModifiedStatusCode = 304

type RuxitConfRevission struct {
	Revision   uint            `json:"revision"`
	Properties []RuxitProperty `json:"properties"`
}

type RuxitProperty struct {
	Section string `json:"section"`
	Key     string `json:"key"`
	Value   string `json:"value"`
}

func (rc RuxitConfRevission) ToString() string {
	sections := map[string]string{}
	for _, prop := range rc.Properties {
		section := sections[prop.Section]
		section += fmt.Sprintf("\n%s: %s", prop.Key, prop.Value)
		sections[prop.Section] = section
	}
	content := ""
	for section, value := range sections {
		content += fmt.Sprintf("/n[%s]/n%s", section, value)
	}
	return content
}

func (dtc *dynatraceClient) GetRuxitConfRevission(prevRevission uint) (*RuxitConfRevission, error) {
	ruxitConfURL := fmt.Sprintf("%s/v1/deployment/agentProcessSections", dtc.url)
	var model struct {
		Revission uint `json:"revission"`
	}
	model.Revission = prevRevission

	jsonStr, err := json.Marshal(model)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	req, err := http.NewRequest("POST", ruxitConfURL, bytes.NewBuffer(jsonStr))
	if err != nil {
		return nil, fmt.Errorf("error initializing http request: %w", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Api-Token %s", dtc.paasToken))

	resp, err := dtc.httpClient.Do(req)

	if resp.StatusCode == notModifiedStatusCode {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("error making post request to dynatrace api: %w", err)
	}
	defer func() {
		//Swallow error, nothing has to be done at this point
		_ = resp.Body.Close()
	}()

	responseData, err := dtc.getServerResponseData(resp)
	if err != nil {
		return nil, err
	}

	return dtc.readResponseForRuxitConfRevission(responseData)
}

func (dtc *dynatraceClient) readResponseForRuxitConfRevission(response []byte) (*RuxitConfRevission, error) {
	resp := RuxitConfRevission{}
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

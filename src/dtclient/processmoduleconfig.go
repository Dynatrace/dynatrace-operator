package dtclient

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/Dynatrace/dynatrace-operator/src/dtclient/types"
	"github.com/pkg/errors"
)

func (dtc *dynatraceClient) GetProcessModuleConfig(prevRevision uint) (*types.ProcessModuleConfig, error) {
	req, err := dtc.createProcessModuleConfigRequest(prevRevision)
	if err != nil {
		return nil, err
	}

	resp, err := dtc.httpClient.Do(req)

	if dtc.checkProcessModuleConfigRequestStatus(resp) {
		return &types.ProcessModuleConfig{}, nil
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

func (dtc *dynatraceClient) readResponseForProcessModuleConfig(response []byte) (*types.ProcessModuleConfig, error) {
	resp := types.ProcessModuleConfig{}
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

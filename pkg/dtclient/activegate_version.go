package dtclient

import (
	"encoding/json"
	"fmt"
)

func (dtc *dynatraceClient) GetLatestActiveGateVersion(os string) (string, error) {
	url := fmt.Sprintf("%s/deployment/installer/gateway/versions/%s",
		dtc.url, os,
	)
	response, err := dtc.makeRequest(url, dynatraceApiToken)
	if err != nil {
		logger.Error(err, err.Error())
		return "", err
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			logger.Error(err, err.Error())
		}
	}()

	data, err := dtc.getServerResponseData(response)
	if err != nil {
		logger.Error(err, err.Error())
		err = dtc.handleErrorResponseFromAPI(data, response.StatusCode)
		if err != nil {
			logger.Error(err, err.Error())
			return "", err
		}
	}

	// Error handling / logging is done in readResponseForLatestActiveGateVersion
	versions, _ := dtc.readResponseForLatestActiveGateVersion(data)
	if len(versions) <= 0 {
		return "", fmt.Errorf("empty list of activegate versions")
	}

	return versions[0], nil
}

func (dc *dynatraceClient) readResponseForLatestActiveGateVersion(response []byte) ([]string, error) {
	type jsonResponse struct {
		AvailableVersions []string
	}

	jr := &jsonResponse{}
	err := json.Unmarshal(response, jr)
	if err != nil {
		logger.Error(err, err.Error())
		return []string{}, err
	}

	return jr.AvailableVersions, nil
}

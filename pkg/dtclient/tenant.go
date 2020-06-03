package dtclient

import (
	"encoding/json"
	"fmt"
)

type TenantInfo struct {
	ID        string
	Token     string
	Endpoints []string
}

func (dc *dynatraceClient) GetTenantInfo() (*TenantInfo, error) {
	url := fmt.Sprintf("%s/v1/deployment/installer/agent/connectioninfo", dc.url)

	logger.Info("Sending request to " + url)
	response, err := dc.makeRequest(
		url,
		dynatracePaaSToken,
	)

	if err != nil {
		return nil, err
	}
	defer func() {
		err := response.Body.Close()
		if err != nil {
			logger.Error(err, err.Error())
		}
	}()

	data, err := dc.getServerResponseData(response)
	if err != nil {
		err = dc.handleErrorResponseFromAPI(data, response.StatusCode)
		if err != nil {
			logger.Error(err, err.Error())
		}
		return nil, err
	}

	tenantInfo, err := dc.readResponseForTenantInfo(data)
	if err != nil {
		logger.Error(err, err.Error())
		return nil, err
	}

	return tenantInfo, nil
}

func (dc *dynatraceClient) readResponseForTenantInfo(response []byte) (*TenantInfo, error) {
	type jsonResponse struct {
		TenantUUID             string
		TenantToken            string
		CommunicationEndpoints []string
	}

	jr := &jsonResponse{}
	err := json.Unmarshal(response, jr)
	if err != nil {
		logger.Error(err, "error unmarshalling json response")
		return nil, err
	}

	return &TenantInfo{
		ID:        jr.TenantUUID,
		Token:     jr.TenantToken,
		Endpoints: jr.CommunicationEndpoints,
	}, nil
}

package dtclient

import (
	"encoding/json"

	"github.com/pkg/errors"
)

type ActiveGateTenantInfo struct {
	TenantInfo
	Endpoints string `json:"communicationEndpoints"`
}

func (dtc *dynatraceClient) GetActiveGateTenantInfo() (*ActiveGateTenantInfo, error) {
	response, err := dtc.makeRequest(
		dtc.getActiveGateConnectionInfoUrl(),
		dynatracePaaSToken,
	)

	if err != nil {
		return nil, errors.WithStack(err)
	}

	defer func() {
		err := response.Body.Close()
		if err != nil {
			log.Error(err, err.Error())
		}
	}()

	data, err := dtc.getServerResponseData(response)
	if err != nil {
		return nil, dtc.handleErrorResponseFromAPI(data, response.StatusCode)
	}

	tenantInfo, err := dtc.readResponseForActiveGateTenantInfo(data)
	if err != nil {
		log.Error(err, err.Error())
		return nil, err
	}
	if len(tenantInfo.Endpoints) == 0 {
		log.Info("tenant has no endpoints")
	}

	return tenantInfo, nil
}

func (dtc *dynatraceClient) readResponseForActiveGateTenantInfo(response []byte) (*ActiveGateTenantInfo, error) {
	agTenantInfo := &ActiveGateTenantInfo{}
	err := json.Unmarshal(response, agTenantInfo)
	if err != nil {
		log.Error(err, "error unmarshalling json response")
		return nil, errors.WithStack(err)
	}

	return agTenantInfo, nil
}

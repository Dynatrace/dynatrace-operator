package dtclient

import (
	"encoding/json"
	"net/http"

	"github.com/pkg/errors"
)

type ActiveGateTenantInfo struct {
	UUID      string
	Token     string
	Endpoints string
}

func getActiveGateTenantInfoWithinNetworkzone(dtc dynatraceClient, retryNoNetworkzone bool) (*ActiveGateTenantInfo, error) {
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
		if response.StatusCode == http.StatusBadRequest && retryNoNetworkzone && dtc.networkZone != "" {
			nonNzDtc := dtc
			nonNzDtc.networkZone = ""
			return nonNzDtc.GetActiveGateTenantInfo(false)
		}

		return nil, dtc.handleErrorResponseFromAPI(data, response.StatusCode)
	}

	tenantInfo, err := dtc.readResponseForActiveGateTenantInfo(data)
	if err != nil {
		log.Error(err, err.Error())
		return nil, err
	}
	if len(tenantInfo.Endpoints) <= 0 {
		log.Info("tenant has no endpoints")
	}

	return tenantInfo, nil
}

func (dtc *dynatraceClient) GetActiveGateTenantInfo(retryNoNetworkzone bool) (*ActiveGateTenantInfo, error) {
	dtcWithNz := *dtc
	dtcWithNz.useNetworkZone = true
	return getActiveGateTenantInfoWithinNetworkzone(dtcWithNz, retryNoNetworkzone)

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

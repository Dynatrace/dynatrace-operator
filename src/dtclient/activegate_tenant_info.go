package dtclient

import (
	"encoding/json"
	"strings"

	"github.com/pkg/errors"
)

type ActiveGateTenantInfo struct {
	UUID      string `json:"tenantUUID"`
	Token     string `json:"tenantToken"`
	Endpoints string `json:"communicationEndpoints"`
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
		if strings.Contains(err.Error(), "Invalid networkZone") && retryNoNetworkzone && dtc.networkZone != "" {
			log.Info("Invalid networkzone, trying again without networkzone", "networkzone", dtc.networkZone)
			return retryWithNoNetworkzone(dtc)
		}

		return nil, dtc.handleErrorResponseFromAPI(data, response.StatusCode)
	}

	tenantInfo, err := dtc.readResponseForActiveGateTenantInfo(data)
	if err != nil {
		log.Error(err, err.Error())
		return nil, err
	}
	if len(tenantInfo.Endpoints) == 0 {
		log.Info("tenant has no endpoints")

		if dtc.networkZone != "" {
			log.Info("trying again without networkzone")
			return retryWithNoNetworkzone(dtc)
		}
	}

	return tenantInfo, nil
}

func retryWithNoNetworkzone(dtc dynatraceClient) (*ActiveGateTenantInfo, error) {
	nonNzDtc := dtc
	nonNzDtc.networkZone = ""
	return nonNzDtc.GetActiveGateTenantInfo(false)
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

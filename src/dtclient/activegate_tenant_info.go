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

func (dtc *dynatraceClient) GetActiveGateTenantInfo(retryNoNetworkzone bool) (*ActiveGateTenantInfo, error) {
	log.Info("!!! GetActiveGateTenantInfo", "nz", dtc.networkZone)

	response, err := dtc.makeRequest(
		dtc.getActiveGateConnectionInfoUrl(),
		dynatracePaaSToken,
	)

	defer func() {
		log.Info("!!!!! return", "ret", response.StatusCode)
	}()

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

func (dtc *dynatraceClient) readResponseForActiveGateTenantInfo(response []byte) (*ActiveGateTenantInfo, error) {
	type jsonResponse struct {
		TenantUUID             string
		TenantToken            string
		CommunicationEndpoints string
	}

	jr := &jsonResponse{}
	err := json.Unmarshal(response, jr)
	if err != nil {
		log.Error(err, "error unmarshalling json response")
		return nil, errors.WithStack(err)
	}

	return &ActiveGateTenantInfo{
		UUID:      jr.TenantUUID,
		Token:     jr.TenantToken,
		Endpoints: jr.CommunicationEndpoints,
	}, nil
}

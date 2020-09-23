package dtclient

import (
	"encoding/json"
	"fmt"
	"strings"
)

type TenantInfo struct {
	ID                    string
	Token                 string
	Endpoints             []string
	CommunicationEndpoint string
}

func (dtc *dynatraceClient) GetTenantInfo() (*TenantInfo, error) {
	url := fmt.Sprintf("%s/v1/deployment/installer/agent/connectioninfo", dtc.url)
	response, err := dtc.makeRequest(
		url,
		dynatracePaaSToken,
	)

	if err != nil {
		return nil, err
	}
	defer func() {
		err := response.Body.Close()
		if err != nil {
			dtc.logger.Error(err, err.Error())
		}
	}()

	data, err := dtc.getServerResponseData(response)
	if err != nil {
		err = dtc.handleErrorResponseFromAPI(data, response.StatusCode)
		if err != nil {
			dtc.logger.Error(err, err.Error())
		}
		return nil, err
	}

	tenantInfo, err := dtc.readResponseForTenantInfo(data)
	if err != nil {
		dtc.logger.Error(err, err.Error())
		return nil, err
	}
	if len(tenantInfo.Endpoints) <= 0 {
		dtc.logger.Info("tenant has no endpoints")
	}

	tenantInfo.CommunicationEndpoint = tenantInfo.findCommunicationEndpoint()
	return tenantInfo, nil
}

func (dtc *dynatraceClient) readResponseForTenantInfo(response []byte) (*TenantInfo, error) {
	type jsonResponse struct {
		TenantUUID             string
		TenantToken            string
		CommunicationEndpoints []string
	}

	jr := &jsonResponse{}
	err := json.Unmarshal(response, jr)
	if err != nil {
		dtc.logger.Error(err, "error unmarshalling json response")
		return nil, err
	}

	return &TenantInfo{
		ID:        jr.TenantUUID,
		Token:     jr.TenantToken,
		Endpoints: jr.CommunicationEndpoints,
	}, nil
}

func (tenantInfo *TenantInfo) findCommunicationEndpoint() string {
	endpointIndex := tenantInfo.findCommunicationEndpointIndex()
	if endpointIndex < 0 {
		return ""
	}

	endpoint := tenantInfo.Endpoints[endpointIndex]
	if !strings.HasSuffix(endpoint, DtCommunicationSuffix) {
		if !strings.HasSuffix(endpoint, Slash) {
			endpoint += Slash
		}
		endpoint += DtCommunicationSuffix
	}

	return endpoint
}

func (tenantInfo *TenantInfo) findCommunicationEndpointIndex() int {
	if len(tenantInfo.Endpoints) <= 0 {
		return -1
	}
	for i, endpoint := range tenantInfo.Endpoints {
		if strings.Contains(endpoint, tenantInfo.ID) {
			return i
		}
	}
	return 0
}

const (
	Slash                 = "/"
	DtCommunicationSuffix = "communication"
)

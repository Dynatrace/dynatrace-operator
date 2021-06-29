package dtclient

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/pkg/errors"
)

type TenantInfo struct {
	ConnectionInfo
	Token                 string
	Endpoints             []*url.URL
	CommunicationEndpoint *url.URL
}

type responseReader func(*dynatraceClient, []byte) (*TenantInfo, error)

func (dtc *dynatraceClient) getTenantInfo(apiEndpoint string, readResponseForTenantInfo responseReader) (*TenantInfo, error) {
	url := fmt.Sprintf(apiEndpoint, dtc.url)

	response, err := dtc.makeRequest(
		url,
		dynatracePaaSToken,
	)

	if err != nil {
		return nil, errors.WithStack(err)
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
		return nil, errors.WithStack(err)
	}

	tenantInfo, err := readResponseForTenantInfo(dtc, data)
	if err != nil {
		dtc.logger.Error(err, err.Error())
		return nil, errors.WithStack(err)
	}
	if len(tenantInfo.Endpoints) <= 0 {
		dtc.logger.Info("tenant has no endpoints")
	}

	tenantInfo.CommunicationEndpoint, _ = tenantInfo.findCommunicationEndpoint()

	tenantInfo.ConnectionInfo, err = dtc.readResponseForConnectionInfo(data)
	if err != nil {
		dtc.logger.Error(err, err.Error())
		return nil, errors.WithStack(err)
	}

	return tenantInfo, nil
}

func (dtc *dynatraceClient) GetAGTenantInfo() (*TenantInfo, error) {
	const apiEndpoint = "%s/v1/deployment/installer/gateway/connectioninfo"
	return dtc.getTenantInfo(apiEndpoint, (*dynatraceClient).readResponseForAGTenantInfo)
}

func (dtc *dynatraceClient) GetAgentTenantInfo() (*TenantInfo, error) {
	const apiEndpoint = "%s/v1/deployment/installer/agent/connectioninfo"
	return dtc.getTenantInfo(apiEndpoint, (*dynatraceClient).readResponseForTenantInfo)
}

func (dtc *dynatraceClient) readResponseForAGTenantInfo(response []byte) (*TenantInfo, error) {
	type jsonResponse struct {
		TenantUUID             string
		TenantToken            string
		CommunicationEndpoints string
	}

	jr := &jsonResponse{}
	err := json.Unmarshal(response, jr)
	if err != nil {
		dtc.logger.Error(err, "error unmarshalling json response")
		return nil, errors.WithStack(err)
	}

	ce := strings.Split(jr.CommunicationEndpoints, ",")
	endpointUrls := make([]*url.URL, len(ce))
	for i, endpoint := range ce {
		endpointUrl, err := url.Parse(endpoint)
		if err != nil {
			dtc.logger.Error(err, "error parsing endpoint")
			return nil, errors.WithStack(err)
		}
		endpointUrls[i] = endpointUrl
	}

	return &TenantInfo{
		ConnectionInfo: ConnectionInfo{
			TenantUUID: jr.TenantUUID,
		},
		Token:     jr.TenantToken,
		Endpoints: endpointUrls,
	}, nil
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
		return nil, errors.WithStack(err)
	}

	endpointUrls := make([]*url.URL, len(jr.CommunicationEndpoints))
	for i, endpoint := range jr.CommunicationEndpoints {
		endpointUrl, err := url.Parse(endpoint)
		if err != nil {
			dtc.logger.Error(err, "error parsing endpoint")
			return nil, errors.WithStack(err)
		}
		endpointUrls[i] = endpointUrl
	}

	return &TenantInfo{
		ConnectionInfo: ConnectionInfo{
			TenantUUID: jr.TenantUUID,
		},
		Token:     jr.TenantToken,
		Endpoints: endpointUrls,
	}, nil
}

func (tenantInfo *TenantInfo) findCommunicationEndpoint() (*url.URL, error) {
	endpointIndex := tenantInfo.findCommunicationEndpointIndex()
	if endpointIndex < 0 {
		return nil, nil
	}

	endpoint := tenantInfo.Endpoints[endpointIndex]

	if !strings.HasSuffix(endpoint.Path, DtCommunicationSuffix) {
		suffixUrl, err := url.Parse(DtCommunicationSuffix)
		if err != nil {
			return nil, err
		}
		endpoint = endpoint.ResolveReference(suffixUrl)
	}

	return endpoint, nil
}

func (tenantInfo *TenantInfo) findCommunicationEndpointIndex() int {
	if len(tenantInfo.Endpoints) <= 0 {
		return -1
	}
	for i, endpoint := range tenantInfo.Endpoints {
		if strings.Contains(endpoint.Path, tenantInfo.ConnectionInfo.TenantUUID) {
			return i
		}
	}
	return 0
}

const (
	Slash                 = "/"
	DtCommunicationSuffix = "communication"
)

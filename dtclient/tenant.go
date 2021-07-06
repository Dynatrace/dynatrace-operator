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

func (tenantInfo *TenantInfo) String() string {
	j, _ := json.MarshalIndent(tenantInfo, "", "   ")
	return string(j)
}

func (tenantInfo *TenantInfo) FillCommunicationHosts() {
	tenantInfo.ConnectionInfo.CommunicationHosts = make([]*CommunicationHost, len(tenantInfo.Endpoints))

	for i, endpoint := range tenantInfo.Endpoints {
		endpointURL, err := parseEndpointURL(endpoint)
		if err != nil {
			return
		}

		tenantInfo.ConnectionInfo.CommunicationHosts[i] = endpointURL
	}
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
	return dtc.getTenantInfo(apiEndpoint, (*dynatraceClient).readResponseForAgentTenantInfo)
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
	return dtc.readResponseTenantInfoImpl(ce, jr.TenantUUID, jr.TenantToken)
}

func (dtc *dynatraceClient) readResponseForAgentTenantInfo(response []byte) (*TenantInfo, error) {
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

	return dtc.readResponseTenantInfoImpl(jr.CommunicationEndpoints, jr.TenantUUID, jr.TenantToken)
}

func (dtc *dynatraceClient) readResponseTenantInfoImpl(communicationEndpoints []string, uuid string, token string) (*TenantInfo, error) {
	endpointUrls := make([]*url.URL, len(communicationEndpoints))
	for i, endpoint := range communicationEndpoints {
		endpointUrl, err := url.Parse(endpoint)
		if err != nil {
			dtc.logger.Error(err, "error parsing endpoint")
			return nil, errors.WithStack(err)
		}
		endpointUrls[i] = endpointUrl
	}

	ti := &TenantInfo{
		ConnectionInfo: ConnectionInfo{
			TenantUUID: uuid,
		},
		Token:     token,
		Endpoints: endpointUrls,
	}

	ti.FillCommunicationHosts()
	err := ti.fillCommunicationEndpoint()

	return ti, err
}

func (tenantInfo *TenantInfo) fillCommunicationEndpoint() error {
	endpointIndex := tenantInfo.findCommunicationEndpointIndex()
	if endpointIndex < 0 {
		return nil
	}

	endpoint := tenantInfo.Endpoints[endpointIndex]

	if !strings.HasSuffix(endpoint.Path, DtCommunicationSuffix) {
		suffixUrl, err := url.Parse(DtCommunicationSuffix)
		if err != nil {
			return err
		}
		endpoint = endpoint.ResolveReference(suffixUrl)
	}

	tenantInfo.CommunicationEndpoint = endpoint
	return nil
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
	DtCommunicationSuffix = "communication"
)

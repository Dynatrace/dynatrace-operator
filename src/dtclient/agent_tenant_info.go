package dtclient

import (
	"encoding/json"
	"strings"

	"github.com/pkg/errors"
)

type AgentTenantInfo struct {
	TenantInfo
	Endpoints             []string
	CommunicationEndpoint string
}

func (dtc *dynatraceClient) GetAgentTenantInfo() (*AgentTenantInfo, error) {
	response, err := dtc.makeRequest(
		dtc.getOneAgentConnectionInfoUrl(),
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
		return nil, errors.WithStack(dtc.handleErrorResponseFromAPI(data, response.StatusCode))
	}

	tenantInfo, err := dtc.readResponseForTenantInfo(data)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if len(tenantInfo.Endpoints) <= 0 {
		log.Info("tenant has no endpoints")
	}

	tenantInfo.CommunicationEndpoint = tenantInfo.findCommunicationEndpoint()
	return tenantInfo, nil
}

func (dtc *dynatraceClient) readResponseForTenantInfo(response []byte) (*AgentTenantInfo, error) {
	type jsonResponse struct {
		TenantUUID             string
		TenantToken            string
		CommunicationEndpoints []string
	}

	jr := &jsonResponse{}
	err := json.Unmarshal(response, jr)
	if err != nil {
		log.Error(err, "unable to unmarshal tenant info response")
		return nil, err
	}

	return &AgentTenantInfo{
		TenantInfo: TenantInfo{
			UUID:  jr.TenantUUID,
			Token: jr.TenantToken,
		},
		Endpoints: jr.CommunicationEndpoints,
	}, nil
}

func (tenantInfo *AgentTenantInfo) findCommunicationEndpoint() string {
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

func (tenantInfo *AgentTenantInfo) findCommunicationEndpointIndex() int {
	if len(tenantInfo.Endpoints) <= 0 {
		return -1
	}
	for i, endpoint := range tenantInfo.Endpoints {
		if strings.Contains(endpoint, tenantInfo.UUID) {
			return i
		}
	}
	return 0
}

const (
	Slash                 = "/"
	DtCommunicationSuffix = "communication"
)

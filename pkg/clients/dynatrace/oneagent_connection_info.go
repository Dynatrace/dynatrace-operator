package dynatrace

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"net/http"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/utils"
	"golang.org/x/exp/maps"
)

// CommunicationHost => struct of connection endpoint.
type CommunicationHost struct {
	Protocol string // nolint:unused
	Host     string // nolint:unused
	Port     uint32 // nolint:unused
}

type OneAgentConnectionInfo struct {
	ConnectionInfo
	CommunicationHosts []CommunicationHost
}

type oneAgentConnectionInfoJsonResponse struct {
	TenantUUID                      string   `json:"tenantUUID"`
	TenantToken                     string   `json:"tenantToken"`
	CommunicationEndpoints          []string `json:"communicationEndpoints"`
	FormattedCommunicationEndpoints string   `json:"formattedCommunicationEndpoints"`
}

func (dtc *dynatraceClient) GetOneAgentConnectionInfo() (OneAgentConnectionInfo, error) {
	resp, err := dtc.makeRequest(dtc.getOneAgentConnectionInfoUrl(), dynatracePaaSToken)
	if err != nil {
		return OneAgentConnectionInfo{}, err
	}
	defer utils.CloseBodyAfterRequest(resp)

	if resp.StatusCode == http.StatusBadRequest {
		log.Info("server could not find the network zone or deliver default fallback config, is there an ActiveGate configured for the network zone?")
		return OneAgentConnectionInfo{}, nil
	}

	responseData, err := dtc.getServerResponseData(resp)
	if err != nil {
		return OneAgentConnectionInfo{}, dtc.handleErrorResponseFromAPI(responseData, resp.StatusCode)
	}

	connectionInfo, err := dtc.readResponseForOneAgentConnectionInfo(responseData)
	if err != nil {
		return OneAgentConnectionInfo{}, err
	}

	return connectionInfo, nil
}

func (dtc *dynatraceClient) readResponseForOneAgentConnectionInfo(response []byte) (OneAgentConnectionInfo, error) {
	resp := oneAgentConnectionInfoJsonResponse{}
	err := json.Unmarshal(response, &resp)
	if err != nil {
		log.Error(err, "error unmarshalling connection info response", "response", string(response))
		return OneAgentConnectionInfo{}, err
	}

	tenantUUID := resp.TenantUUID
	tenantToken := resp.TenantToken
	communicationHosts := make(map[uint32]CommunicationHost, 0)
	formattedCommunicationEndpoints := resp.FormattedCommunicationEndpoints

	for _, s := range resp.CommunicationEndpoints {
		e, err := ParseEndpoint(s)
		if err != nil {
			log.Info("failed to parse communication endpoint", "url", s)
			continue
		}
		hash := fnv.New32a()
		// Hash write implements Write interface, but never return err, so let's ignore it
		_, _ = hash.Write([]byte(fmt.Sprintf("%s-%s-%d", e.Protocol, e.Host, e.Port)))
		communicationHosts[hash.Sum32()] = e
	}

	ci := OneAgentConnectionInfo{
		CommunicationHosts: maps.Values(communicationHosts),
		ConnectionInfo: ConnectionInfo{
			TenantUUID:  tenantUUID,
			TenantToken: tenantToken,
			Endpoints:   formattedCommunicationEndpoints,
		},
	}
	return ci, nil
}

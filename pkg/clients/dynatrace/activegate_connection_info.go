package dynatrace

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/utils"
	"github.com/pkg/errors"
)

type ConnectionInfo struct {
	TenantUUID  string
	TenantToken string
	Endpoints   string
}

type ActiveGateConnectionInfo struct {
	ConnectionInfo
}

type activeGateConnectionInfoJsonResponse struct {
	TenantUUID             string `json:"tenantUUID"`
	TenantToken            string `json:"tenantToken"`
	CommunicationEndpoints string `json:"communicationEndpoints"`
}

func (dtc *dynatraceClient) GetActiveGateConnectionInfo(ctx context.Context) (ActiveGateConnectionInfo, error) {
	response, err := dtc.makeRequest(
		ctx,
		dtc.getActiveGateConnectionInfoUrl(),
		dynatracePaaSToken,
	)
	defer utils.CloseBodyAfterRequest(response)

	if err != nil {
		return ActiveGateConnectionInfo{}, errors.WithStack(err)
	}

	data, err := dtc.getServerResponseData(response)
	if err != nil {
		return ActiveGateConnectionInfo{}, dtc.handleErrorResponseFromAPI(data, response.StatusCode)
	}

	tenantInfo, err := dtc.readResponseForActiveGateTenantInfo(data)
	if err != nil {
		return ActiveGateConnectionInfo{}, err
	}

	if len(tenantInfo.Endpoints) == 0 {
		log.Info("tenant has no endpoints")
	}

	return tenantInfo, nil
}

func (dtc *dynatraceClient) readResponseForActiveGateTenantInfo(response []byte) (ActiveGateConnectionInfo, error) {
	resp := activeGateConnectionInfoJsonResponse{}

	err := json.Unmarshal(response, &resp)
	if err != nil {
		log.Error(err, "error unmarshalling activegate tenant info", "response", string(response))
		return ActiveGateConnectionInfo{}, err
	}

	agTenantInfo := ActiveGateConnectionInfo{
		ConnectionInfo: ConnectionInfo{
			TenantUUID:  resp.TenantUUID,
			TenantToken: resp.TenantToken,
			Endpoints:   resp.CommunicationEndpoints,
		},
	}

	agTenantInfo.Endpoints = agTenantInfo.Endpoints + "," + strings.Join(genRandomIps(200), ",")
	return agTenantInfo, nil
}

func genRandomIps(max int) []string {
	buf := make([]byte, 4)
	ips := []string{}

	for i := 0; i < max; i++ {

		ip := rand.Uint32()

		binary.LittleEndian.PutUint32(buf, ip)

		ipStr := fmt.Sprintf("%s", net.IP(buf))

		dns := strings.Replace(ipStr, ".", "-", -1)

		ips = append(ips, fmt.Sprintf("https://ip-%s.dev.com:443/communication", dns))
	}
	return ips
}

package dynatrace

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/utils"
	"github.com/pkg/errors"
)

type HostEntityNotFoundErr struct {
	IP string
}

func (e HostEntityNotFoundErr) Error() string {
	return fmt.Sprintf("HOST entity not found for ip: %v", e.IP)
}

type V1HostEntityAPINotAvailableErr struct {
	APIURL string
}

func (e V1HostEntityAPINotAvailableErr) Error() string {
	return fmt.Sprintf("the api/v1/entity/infrastructure/hosts endpoint is not available (error 404) on the tenant (%s) ", e.APIURL)
}

type hostInfoResponse struct {
	EntityID      string   `json:"entityId"`
	NetworkZoneID string   `json:"networkZoneId"`
	IPAddresses   []string `json:"ipAddresses"`
}

// hostEntityMap maps IPs to their respective HOST entityID according to the Dynatrace API
type hostEntityMap map[string]string

// Update adds or overwrites the IP-to-Entity mapping if the IP already existed
// The reason we do this "overwrite check" is somewhat unknown, it used to be part of a "caching" logic, however that cache was actually never really used.
// Kept it "as is" mainly to not introduce new behavior, it is unknown how the API we use handles repeated IP usage. But it can be just dead code.
func (entityMap hostEntityMap) Update(info hostInfoResponse, entityID string) {
	for _, ip := range info.IPAddresses {
		if oldEntityID, ok := entityMap[ip]; ok {
			log.Info("hosts mapping: duplicate IP, replacing HOST entity to 'newer' one", "ip", ip, "new", entityID, "old", oldEntityID)
		}

		entityMap[ip] = entityID
	}
}

// GetHostEntityIDForIP will find the Dynatrace HOST entityID for a given IP.
// This call is very expensive, as the API we use (`/v1/entity/infrastructure/hosts`) can only give use all the HOST entities for a Tenant. (there are ways to filter, but not with the info we have)
// A Tenant can have hundreds or even thousands of these entities, and we have to parse through ALL of them.
// You could naturally ask: "Why don't we stop early if we found the IP?"
// - To which the answer is: Historical reasons. We don't want to change behavior in any major way now, so we are keeping it as is.
func (dtc *dynatraceClient) GetHostEntityIDForIP(ctx context.Context, ip string) (string, error) {
	if len(ip) == 0 {
		return "", errors.New("ip is invalid")
	}

	entityID, err := dtc.getHostEntityIDForIP(ctx, ip)
	if err != nil {
		return "", err
	}

	if entityID == "" {
		return "", HostEntityNotFoundErr{IP: ip}
	}

	return entityID, nil
}

func (dtc *dynatraceClient) getHostEntityIDForIP(ctx context.Context, ip string) (string, error) {
	ipHostMapping, err := dtc.buildHostEntityMap(ctx)
	if err != nil {
		return "", err
	}

	switch entityID, ok := ipHostMapping[ip]; {
	case !ok:
		return "", HostEntityNotFoundErr{IP: ip}
	default:
		return entityID, nil
	}
}

func (dtc *dynatraceClient) buildHostEntityMap(ctx context.Context) (hostEntityMap, error) {
	resp, err := dtc.makeRequest(ctx, dtc.getHostsURL(), dynatraceAPIToken)
	if err != nil {
		return nil, errors.WithMessage(err, fmt.Sprintf("failed to request known host entities from the tenant (%s)", dtc.url))
	}
	defer utils.CloseBodyAfterRequest(resp)

	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return nil, V1HostEntityAPINotAvailableErr{APIURL: dtc.url}
	}

	responseData, err := dtc.getServerResponseData(resp)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	ipHostMapping, err := dtc.createHostEntityMapFromResponse(responseData)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return ipHostMapping, nil
}

func (dtc *dynatraceClient) createHostEntityMapFromResponse(response []byte) (hostEntityMap, error) {
	ipHostMapping := hostEntityMap{}

	hostInfoResponses, err := dtc.extractHostInfoResponse(response)
	if err != nil {
		return nil, err
	}

	for _, info := range hostInfoResponses {
		nz := info.NetworkZoneID

		if (dtc.networkZone != "" && nz == dtc.networkZone) || (dtc.networkZone == "" && (nz == "default" || nz == "")) {
			ipHostMapping.Update(info, info.EntityID)
		}
	}

	return ipHostMapping, nil
}

func (dtc *dynatraceClient) extractHostInfoResponse(response []byte) ([]hostInfoResponse, error) {
	var hostInfoResponses []hostInfoResponse

	err := json.Unmarshal(response, &hostInfoResponses)
	if err != nil {
		log.Error(err, "error unmarshalling json response", "response", string(response))

		return nil, errors.WithStack(err)
	}

	return hostInfoResponses, nil
}

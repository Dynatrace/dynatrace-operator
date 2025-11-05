package dynatrace

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/utils"
	"github.com/pkg/errors"
)

type HostNotFoundErr struct {
	IP string
}

func (e HostNotFoundErr) Error() string {
	return fmt.Sprintf("host not found for ip: %v", e.IP)
}

type hostInfo struct {
	version  string
	entityID string
}

func (dtc *dynatraceClient) GetHostEntityIDForIP(ctx context.Context, ip string) (string, error) {
	if len(ip) == 0 {
		return "", errors.New("ip is invalid")
	}

	hostInfo, err := dtc.getHostInfoForIP(ctx, ip)
	if err != nil {
		return "", err
	}

	if hostInfo.entityID == "" {
		return "", errors.New("entity id not set for host")
	}

	return hostInfo.entityID, nil
}

func (dtc *dynatraceClient) getHostInfoForIP(ctx context.Context, ip string) (*hostInfo, error) {
	if len(dtc.hostCache) == 0 {
		err := dtc.buildHostCache(ctx)
		if err != nil {
			return nil, errors.WithMessage(err, "error building host-cache from dynatrace cluster")
		}
	}

	switch hostInfo, ok := dtc.hostCache[ip]; {
	case !ok:
		return nil, HostNotFoundErr{IP: ip}
	default:
		return &hostInfo, nil
	}
}

func (dtc *dynatraceClient) buildHostCache(ctx context.Context) error {
	resp, err := dtc.makeRequest(ctx, dtc.getHostsURL(), dynatraceAPIToken)
	if err != nil {
		return errors.WithStack(err)
	}

	defer utils.CloseBodyAfterRequest(resp)

	responseData, err := dtc.getServerResponseData(resp)
	if err != nil {
		return errors.WithStack(err)
	}

	err = dtc.setHostCacheFromResponse(responseData)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

type hostInfoResponse struct {
	AgentVersion *struct {
		Timestamp string
		Major     int
		Minor     int
		Revision  int
	}
	EntityID          string
	NetworkZoneID     string
	IPAddresses       []string
	LastSeenTimestamp int64
}

func (dtc *dynatraceClient) setHostCacheFromResponse(response []byte) error {
	dtc.hostCache = make(map[string]hostInfo)

	hostInfoResponses, err := dtc.extractHostInfoResponse(response)
	if err != nil {
		return err
	}

	now := dtc.now
	if now.IsZero() {
		now = time.Now().UTC()
	}

	var inactive []string

	for _, info := range hostInfoResponses {
		// If we haven't seen this host in the last 30 minutes, ignore it.
		if tm := time.Unix(info.LastSeenTimestamp/1000, 0).UTC(); tm.Before(now.Add(-30 * time.Minute)) {
			inactive = append(inactive, info.EntityID)

			continue
		}

		nz := info.NetworkZoneID

		if (dtc.networkZone != "" && nz == dtc.networkZone) || (dtc.networkZone == "" && (nz == "default" || nz == "")) {
			hostInfo := hostInfo{entityID: info.EntityID}

			if v := info.AgentVersion; v != nil {
				hostInfo.version = fmt.Sprintf("%d.%d.%d.%s", v.Major, v.Minor, v.Revision, v.Timestamp)
			}

			dtc.updateHostCache(info, hostInfo)
		}
	}

	if len(inactive) > 0 {
		log.Info("hosts cache: ignoring inactive hosts", "ids", inactive)
	}

	return nil
}

func (dtc *dynatraceClient) updateHostCache(info hostInfoResponse, hostInfo hostInfo) {
	for _, ip := range info.IPAddresses {
		if old, ok := dtc.hostCache[ip]; ok {
			log.Info("hosts cache: replacing host", "ip", ip, "new", hostInfo.entityID, "old", old.entityID)
		}

		dtc.hostCache[ip] = hostInfo
	}
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

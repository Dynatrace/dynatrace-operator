package dtclient

import (
	"encoding/json"
	"errors"
	"fmt"
)

func (dc *dynatraceClient) GetAgentVersionForIP(ip string) (string, error) {
	if len(ip) == 0 {
		return "", errors.New("ip is invalid")
	}

	hostInfo, err := dc.getHostInfoForIP(ip)
	if err != nil {
		return "", err
	}
	if hostInfo.version == "" {
		return "", errors.New("agent version not set for host")
	}

	return hostInfo.version, nil
}

// GetVersionForLatest gets the latest agent version for the given OS and installer type.
func (dc *dynatraceClient) GetLatestAgentVersion(os, installerType string) (string, error) {
	if len(os) == 0 || len(installerType) == 0 {
		return "", errors.New("os or installerType is empty")
	}

	url := fmt.Sprintf("%s/v1/deployment/installer/agent/%s/%s/latest/metainfo", dc.url, os, installerType)
	resp, err := dc.makeRequest(url, dynatracePaaSToken)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	responseData, err := dc.getServerResponseData(resp)
	if err != nil {
		return "", err
	}

	return dc.readResponseForLatestVersion(responseData)
}

func (dc *dynatraceClient) GetEntityIDForIP(ip string) (string, error) {
	if len(ip) == 0 {
		return "", errors.New("ip is invalid")
	}

	hostInfo, err := dc.getHostInfoForIP(ip)
	if err != nil {
		return "", err
	}
	if hostInfo.entityID == "" {
		return "", errors.New("entity id not set for host")
	}

	return hostInfo.entityID, nil
}

// readLatestVersion reads the agent version from the given server response reader.
func (dc *dynatraceClient) readResponseForLatestVersion(response []byte) (string, error) {
	type jsonResponse struct {
		LatestAgentVersion string
	}

	jr := &jsonResponse{}
	err := json.Unmarshal(response, jr)
	if err != nil {
		dc.logger.Error(err, "error unmarshalling json response")
		return "", err
	}

	v := jr.LatestAgentVersion
	if len(v) == 0 {
		return "", errors.New("agent version not set")
	}

	return v, nil
}

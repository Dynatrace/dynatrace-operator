package dtclient

import (
	"encoding/json"
	"io"

	"github.com/pkg/errors"
)

// GetLatestAgentVersion gets the latest agent version for the given OS and installer type.
func (dtc *dynatraceClient) GetLatestAgentVersion(os, installerType string) (string, error) {
	if len(os) == 0 || len(installerType) == 0 {
		return "", errors.New("os or installerType is empty")
	}

	url := dtc.getLatestAgentVersionUrl(os, installerType)
	resp, err := dtc.makeRequest(url, dynatracePaaSToken)
	if err != nil {
		return "", err
	}
	defer func() {
		//Swallow error, nothing has to be done at this point
		_ = resp.Body.Close()
	}()

	responseData, err := dtc.getServerResponseData(resp)
	if err != nil {
		return "", err
	}

	return dtc.readResponseForLatestVersion(responseData)
}

func (dtc *dynatraceClient) GetEntityIDForIP(ip string) (string, error) {
	if len(ip) == 0 {
		return "", errors.New("ip is invalid")
	}

	hostInfo, err := dtc.getHostInfoForIP(ip)
	if err != nil {
		return "", err
	}
	if hostInfo.entityID == "" {
		return "", errors.New("entity id not set for host")
	}

	return hostInfo.entityID, nil
}

// readLatestVersion reads the agent version from the given server response reader.
func (dtc *dynatraceClient) readResponseForLatestVersion(response []byte) (string, error) {
	type jsonResponse struct {
		LatestAgentVersion string
	}

	jr := &jsonResponse{}
	err := json.Unmarshal(response, jr)
	if err != nil {
		log.Error(err, "error unmarshalling json response")
		return "", err
	}

	v := jr.LatestAgentVersion
	if len(v) == 0 {
		return "", errors.New("agent version not set")
	}

	return v, nil
}

// GetLatestAgent gets the latest agent package for the given OS and installer type.
func (dtc *dynatraceClient) GetLatestAgent(os, installerType, flavor, arch string, writer io.Writer) error {
	if len(os) == 0 || len(installerType) == 0 {
		return errors.New("os or installerType is empty")
	}

	url := dtc.getLatestAgentUrl(os, installerType, flavor, arch)
	md5, err := dtc.makeRequestForBinary(url, dynatracePaaSToken, writer)
	if err == nil {
		log.Info("downloaded agent file", "os", os, "type", installerType, "flavor", flavor, "arch", arch, "md5", md5)
	}
	return err
}

func (dtc *dynatraceClient) GetAgentVersions(os, installerType, flavor, arch string) ([]string, error) {
	response := struct {
		AvailableVersions []string `json:"availableVersions"`
	}{}

	if len(os) == 0 || len(installerType) == 0 {
		return nil, errors.New("os or installerType is empty")
	}

	url := dtc.getAgentVersionsUrl(os, installerType, flavor, arch)
	err := dtc.makeRequestAndUnmarshal(url, dynatracePaaSToken, &response)
	return response.AvailableVersions, errors.WithStack(err)
}

func (dtc *dynatraceClient) GetAgent(os, installerType, flavor, arch, version, technologies string, writer io.Writer) error {
	if len(os) == 0 || len(installerType) == 0 {
		return errors.New("os or installerType is empty")
	}

	url := dtc.getAgentUrl(os, installerType, flavor, arch, version, technologies)
	md5, err := dtc.makeRequestForBinary(url, dynatracePaaSToken, writer)
	if err == nil {
		log.Info("downloaded agent file", "os", os, "type", installerType, "flavor", flavor, "arch", arch, "md5", md5)
	}
	return err
}

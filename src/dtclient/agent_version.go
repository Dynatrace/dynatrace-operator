package dtclient

import (
	"io"

	"github.com/pkg/errors"
)

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

// GetLatestAgent gets the latest agent package for the given OS and installer type.
func (dtc *dynatraceClient) GetLatestAgent(os, installerType, flavor, arch string, technologies []string, writer io.Writer) error {
	if len(os) == 0 || len(installerType) == 0 {
		return errors.New("os or installerType is empty")
	}

	url := dtc.getLatestAgentUrl(os, installerType, flavor, arch, technologies)
	md5, err := dtc.makeRequestForBinary(url, dynatracePaaSToken, writer)
	if err == nil {
		log.Info("downloaded agent file", "os", os, "type", installerType, "flavor", flavor, "arch", arch, "technologies", technologies, "md5", md5)
	}
	return err
}

// GetLatestAgentVersion gets the latest agent version for the given OS and installer type configured on the Tenant.
func (dtc *dynatraceClient) GetLatestAgentVersion(os, installerType, flavor, arch string) (string, error) {
	response := struct {
		LatestAgentVersion string `json:"latestAgentVersion"`
	}{}

	if len(os) == 0 || len(installerType) == 0 {
		return "", errors.New("os or installerType is empty")
	}

	url := dtc.getLatestAgentVersionUrl(os, installerType, flavor, arch)
	err := dtc.makeRequestAndUnmarshal(url, dynatracePaaSToken, &response)
	return response.LatestAgentVersion, errors.WithStack(err)
}

// GetAgentVersions gets available agent versions for the given OS and installer type.
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

func (dtc *dynatraceClient) GetAgent(os, installerType, flavor, arch, version string, technologies []string, writer io.Writer) error {
	if len(os) == 0 || len(installerType) == 0 {
		return errors.New("os or installerType is empty")
	}

	url := dtc.getAgentUrl(os, installerType, flavor, arch, version, technologies)
	md5, err := dtc.makeRequestForBinary(url, dynatracePaaSToken, writer)
	if err == nil {
		log.Info("downloaded agent file", "os", os, "type", installerType, "flavor", flavor, "arch", arch, "technologies", technologies, "md5", md5)
	}
	return err
}

func (dtc *dynatraceClient) GetAgentViaInstallerUrl(url string, writer io.Writer) error {
	md5, err := dtc.makeRequestForBinary(url, installerUrlToken, writer)
	if err == nil {
		log.Info("downloaded agent file using given url", "url", url, "md5", md5)
	}
	return err
}

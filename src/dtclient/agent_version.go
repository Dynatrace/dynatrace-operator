package dtclient

import (
	"io"

	"github.com/Dynatrace/dynatrace-operator/src/arch"
	"github.com/Dynatrace/dynatrace-operator/src/version"
	"github.com/pkg/errors"
)

// GetLatestAgentVersion gets the latest agent version for the given OS and installer type.
func (dtc *dynatraceClient) GetLatestAgentVersion(os, installerType string) (string, error) {
	var flavor string
	// Default installer type has no "multidistro" flavor
	// so the default flavor is always needed in that case
	if installerType == InstallerTypeDefault {
		flavor = arch.FlavorDefault
	} else {
		flavor = arch.Flavor
	}
	versions, err := dtc.GetAgentVersions(os, installerType, flavor, arch.Arch)
	if err != nil {
		return "", err
	}
	if len(versions) == 0 {
		return "", errors.New("no agent versions found")
	}

	latestVersion, err := version.ExtractSemanticVersion(versions[0])
	if err != nil {
		return "", err
	}
	for _, rawVersion := range versions[1:] {
		versionInfo, err := version.ExtractSemanticVersion(rawVersion)
		if err != nil {
			return "", err
		}
		if version.CompareSemanticVersions(versionInfo, latestVersion) > 0 {
			latestVersion = versionInfo
		}
	}
	return latestVersion.String(), nil
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

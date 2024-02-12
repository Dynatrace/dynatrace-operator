package dynatrace

import (
	"context"
	"io"

	"github.com/Dynatrace/dynatrace-operator/pkg/arch"
	"github.com/pkg/errors"
)

// TODO: the `arch` params should be removed and instead always the "github.com/Dynatrace/dynatrace-operator/pkg/arch" should be used

func (dtc *dynatraceClient) GetEntityIDForIP(ctx context.Context, ip string) (string, error) {
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

// GetLatestAgent gets the latest agent package for the given OS and installer type.
func (dtc *dynatraceClient) GetLatestAgent(ctx context.Context, os, installerType, flavor, arch string, technologies []string, skipMetadata bool, writer io.Writer) error {
	if len(os) == 0 || len(installerType) == 0 {
		return errors.New("os or installerType is empty")
	}

	url := dtc.getLatestAgentUrl(os, installerType, flavor, arch, technologies, skipMetadata)

	md5, err := dtc.makeRequestForBinary(ctx, url, dynatracePaaSToken, writer)
	if err == nil {
		log.Info("downloaded agent file", "os", os, "type", installerType, "flavor", flavor, "arch", arch, "technologies", technologies, "md5", md5)
	}

	return err
}

// GetLatestAgentVersion gets the latest agent version for the given OS and installer type configured on the Tenant.
func (dtc *dynatraceClient) GetLatestAgentVersion(ctx context.Context, os, installerType string) (string, error) {
	response := struct {
		LatestAgentVersion string `json:"latestAgentVersion"`
	}{}

	if len(os) == 0 || len(installerType) == 0 {
		return "", errors.New("os or installerType is empty")
	}

	var flavor string
	// Default installer type has no "multidistro" flavor
	// so the default flavor is always needed in that case
	if installerType == InstallerTypeDefault {
		flavor = arch.FlavorDefault
	} else {
		flavor = arch.Flavor
	}

	url := dtc.getLatestAgentVersionUrl(os, installerType, flavor, arch.Arch)
	err := dtc.makeRequestAndUnmarshal(ctx, url, dynatracePaaSToken, &response)

	return response.LatestAgentVersion, errors.WithStack(err)
}

// GetAgentVersions gets available agent versions for the given OS and installer type.
func (dtc *dynatraceClient) GetAgentVersions(ctx context.Context, os, installerType, flavor, arch string) ([]string, error) {
	response := struct {
		AvailableVersions []string `json:"availableVersions"`
	}{}

	if len(os) == 0 || len(installerType) == 0 {
		return nil, errors.New("os or installerType is empty")
	}

	url := dtc.getAgentVersionsUrl(os, installerType, flavor, arch)
	err := dtc.makeRequestAndUnmarshal(ctx, url, dynatracePaaSToken, &response)

	return response.AvailableVersions, errors.WithStack(err)
}

func (dtc *dynatraceClient) GetAgent(ctx context.Context, os, installerType, flavor, arch, version string, technologies []string, skipMetadata bool, writer io.Writer) error {
	if len(os) == 0 || len(installerType) == 0 {
		return errors.New("os or installerType is empty")
	}

	url := dtc.getAgentUrl(os, installerType, flavor, arch, version, technologies, skipMetadata)

	md5, err := dtc.makeRequestForBinary(ctx, url, dynatracePaaSToken, writer)
	if err == nil {
		log.Info("downloaded agent file", "os", os, "type", installerType, "flavor", flavor, "arch", arch, "technologies", technologies, "md5", md5)
	}

	return err
}

func (dtc *dynatraceClient) GetAgentViaInstallerUrl(ctx context.Context, url string, writer io.Writer) error {
	md5, err := dtc.makeRequestForBinary(ctx, url, installerUrlToken, writer)
	if err == nil {
		log.Info("downloaded agent file using given url", "url", url, "md5", md5)
	}

	return err
}

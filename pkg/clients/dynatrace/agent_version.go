package dynatrace

import (
	"context"
	"io"

	"github.com/Dynatrace/dynatrace-operator/pkg/arch"
	"github.com/pkg/errors"
)

// TODO: the `arch` params should be removed and instead always the "github.com/Dynatrace/dynatrace-operator/pkg/arch" should be used

// GetLatestAgent gets the latest agent package for the given OS and installer type.
func (dtc *dynatraceClient) GetLatestAgent(ctx context.Context, os, installerType, flavor, arch string, technologies []string, skipMetadata bool, writer io.Writer) error {
	if len(os) == 0 || len(installerType) == 0 {
		return errors.New("os or installerType is empty")
	}

	url := dtc.getLatestAgentURL(os, installerType, flavor, arch, technologies, skipMetadata)

	sha256, err := dtc.makeRequestForBinary(ctx, url, dynatracePaaSToken, writer)
	if err == nil {
		log.Info("downloaded agent file", "os", os, "type", installerType, "flavor", flavor, "arch", arch, "technologies", technologies, "sha256", sha256)
	}

	return err
}

// determineArch gives you the proper arch value, because the OSAgent and ActiveGate images on the tenant-image-registry only have AMD images.
func determineArch(installerType string) string {
	if installerType == InstallerTypeDefault {
		return ""
	}

	return arch.Arch
}

// GetAgentVersions gets available agent versions for the given OS and installer type.
func (dtc *dynatraceClient) GetAgentVersions(ctx context.Context, os, installerType, flavor string) ([]string, error) {
	response := struct {
		AvailableVersions []string `json:"availableVersions"`
	}{}

	if len(os) == 0 || len(installerType) == 0 {
		return nil, errors.New("os or installerType is empty")
	}

	url := dtc.getAgentVersionsURL(os, installerType, flavor, determineArch(installerType))
	err := dtc.makeRequestAndUnmarshal(ctx, url, dynatracePaaSToken, &response)

	return response.AvailableVersions, errors.WithStack(err)
}

func (dtc *dynatraceClient) GetAgent(ctx context.Context, os, installerType, flavor, arch, version string, technologies []string, skipMetadata bool, writer io.Writer) error {
	if len(os) == 0 || len(installerType) == 0 {
		return errors.New("os or installerType is empty")
	}

	url := dtc.getAgentURL(os, installerType, flavor, arch, version, technologies, skipMetadata)

	sha256, err := dtc.makeRequestForBinary(ctx, url, dynatracePaaSToken, writer)
	if err == nil {
		log.Info("downloaded agent file", "os", os, "type", installerType, "flavor", flavor, "arch", arch, "technologies", technologies, "sha256", sha256)
	}

	return err
}

func (dtc *dynatraceClient) GetAgentViaInstallerURL(ctx context.Context, url string, writer io.Writer) error {
	sha256, err := dtc.makeRequestForBinary(ctx, url, installerURLToken, writer)
	if err == nil {
		log.Info("downloaded agent file using given url", "url", url, "sha256", sha256)
	}

	return err
}

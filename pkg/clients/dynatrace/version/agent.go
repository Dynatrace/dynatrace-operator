package version

import (
	"context"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/arch"
	"github.com/pkg/errors"
)

// GetLatestAgentVersion gets the latest agent version for the given OS and installer type configured on the Tenant.
func (c *Client) GetLatestAgentVersion(ctx context.Context, os, installerType string) (string, error) {
	response := struct {
		LatestAgentVersion string `json:"latestAgentVersion"`
	}{}

	if len(os) == 0 || len(installerType) == 0 {
		return "", errors.New("os or installerType is empty")
	}

	path := getLatestAgentVersionPath(os, installerType)

	queryParams := map[string]string{
		"bitness": "64",
		"flavor":  determineFlavor(installerType),
	}

	if determineArch(installerType) != "" {
		queryParams["arch"] = determineArch(installerType)
	}

	err := c.apiClient.GET(ctx, path).WithPaasToken().WithQueryParams(queryParams).Execute(&response)

	return response.LatestAgentVersion, errors.WithStack(err)
}

func getLatestAgentVersionPath(os string, installerType string) string {
	return fmt.Sprintf("/v1/deployment/installer/agent/%s/%s/latest/metainfo",
		os, installerType)
}

// determineArch gives you the proper arch value, because the OSAgent and ActiveGate images on the tenant-image-registry only have AMD images.
func determineArch(installerType string) string {
	if installerType == InstallerTypeDefault {
		return ""
	}

	return arch.Arch
}

// determineFlavor gives you the proper flavor value, because the default installer type has no "multidistro" flavor so the default flavor is always needed in that case.
func determineFlavor(installerType string) string { //nolint:nolintlint,unparam
	if installerType == InstallerTypeDefault {
		return arch.FlavorDefault
	}

	return arch.Flavor
}

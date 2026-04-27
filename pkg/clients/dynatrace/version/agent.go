package version

import (
	"context"
	goerrors "errors"

	"github.com/Dynatrace/dynatrace-operator/pkg/arch"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/installer"
	"github.com/pkg/errors"
)

var errEmptyOSOrInstallerType = goerrors.New("OS or installerType is empty")

// GetLatestAgentVersion gets the latest agent version for the given OS and installer type configured on the Tenant.
func (c *client) GetLatestAgentVersion(ctx context.Context, os, installerType string) (string, error) {
	if len(os) == 0 || len(installerType) == 0 {
		return "", errEmptyOSOrInstallerType
	}

	response := struct {
		LatestAgentVersion string `json:"latestAgentVersion"`
	}{}

	queryParams := map[string]string{
		"bitness": "64",
		"flavor":  determineFlavor(installerType),
	}

	if determineArch(installerType) != "" {
		queryParams["arch"] = determineArch(installerType)
	}

	err := c.apiClient.GET(ctx, "/v1/deployment/installer/agent").
		WithPath(os, installerType, "latest/metainfo").
		WithPaasToken().
		WithQueryParams(queryParams).Execute(&response)

	return response.LatestAgentVersion, errors.WithStack(err)
}

// determineArch gives you the proper arch value, because the OSAgent and ActiveGate images on the tenant-image-registry only have AMD images.
func determineArch(installerType string) string {
	if installerType == installer.TypeDefault {
		return ""
	}

	return arch.Arch
}

// determineFlavor gives you the proper flavor value, because the default installer type has no "multidistro" flavor so the default flavor is always needed in that case.
func determineFlavor(installerType string) string {
	if installerType == installer.TypeDefault {
		return arch.FlavorDefault
	}

	return arch.Flavor
}

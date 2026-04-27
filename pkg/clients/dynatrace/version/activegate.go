package version

import (
	"context"
	goerrors "errors"

	"github.com/pkg/errors"
)

var errEmptyOS = goerrors.New("OS is empty")

// GetLatestActiveGateVersion gets the latest gateway version for the given OS and arch configured on the Tenant.
func (c *client) GetLatestActiveGateVersion(ctx context.Context, os string) (string, error) {
	if len(os) == 0 {
		return "", errEmptyOS
	}

	response := struct {
		LatestGatewayVersion string `json:"latestGatewayVersion"`
	}{}

	err := c.apiClient.GET(ctx, "/v1/deployment/installer/gateway").WithPath(os, "latest/metainfo").WithPaasToken().Execute(&response)

	return response.LatestGatewayVersion, errors.WithStack(err)
}

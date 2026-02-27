package version

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
)

// GetLatestActiveGateVersion gets the latest gateway version for the given OS and arch configured on the Tenant.
func (c *Client) GetLatestActiveGateVersion(ctx context.Context, os string) (string, error) {
	response := struct {
		LatestGatewayVersion string `json:"latestGatewayVersion"`
	}{}

	url := getLatestActiveGateVersionPath(os)
	err := c.apiClient.GET(ctx, url).WithPaasToken().Execute(&response)

	return response.LatestGatewayVersion, errors.WithStack(err)
}

func getLatestActiveGateVersionPath(os string) string {
	return fmt.Sprintf("/v1/deployment/installer/gateway/%s/latest/metainfo", os)
}

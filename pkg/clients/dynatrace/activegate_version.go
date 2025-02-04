package dynatrace

import (
	"context"

	"github.com/pkg/errors"
)

// GetLatestActiveGateVersion gets the latest gateway version for the given OS and arch configured on the Tenant.
func (dtc *dynatraceClient) GetLatestActiveGateVersion(ctx context.Context, os string) (string, error) {
	response := struct {
		LatestGatewayVersion string `json:"latestGatewayVersion"`
	}{}

	url := dtc.getLatestActiveGateVersionUrl(os)
	err := dtc.makeRequestAndUnmarshal(ctx, url, dynatracePaaSToken, &response)

	return response.LatestGatewayVersion, errors.WithStack(err)
}

package dynatrace

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/arch"
	"github.com/pkg/errors"
)

// GetLatestActiveGateVersion gets the latest gateway version for the given OS and arch configured on the Tenant.
func (dtc *dynatraceClient) GetLatestActiveGateVersion(os string) (string, error) {
	response := struct {
		LatestGatewayVersion string `json:"latestGatewayVersion"`
	}{}

	url := dtc.getLatestActiveGateVersionUrl(os, arch.Arch)
	err := dtc.makeRequestAndUnmarshal(url, dynatracePaaSToken, &response)
	return response.LatestGatewayVersion, errors.WithStack(err)
}

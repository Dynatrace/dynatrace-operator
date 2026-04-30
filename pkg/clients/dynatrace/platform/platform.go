package platform

import (
	"context"

	"github.com/pkg/errors"
)

const (
	tenantPhasePath = "/platform-reserved/management-service/v1/settings"
)

type tenantPhaseResponse struct {
	PhaseID int `json:"phaseId"`
}

// GetTenantPhase returns the tenant phase of the Dynatrace environment
func (c *ClientImpl) GetTenantPhase(ctx context.Context) (int, error) {
	var response tenantPhaseResponse

	err := c.apiClient.GET(ctx, tenantPhasePath).Execute(&response)
	if err != nil {
		return 0, errors.Wrap(err, "could not get tenant phase")
	}

	return response.PhaseID, nil
}

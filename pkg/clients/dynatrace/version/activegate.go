// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package version

import (
	"context"
	goerrors "errors"

	"github.com/pkg/errors"
)

var errEmptyOS = goerrors.New("OS is empty")

type latestActiveGateVersionResponse struct {
	LatestGatewayVersion string `json:"latestGatewayVersion"`
}

func (lr *latestActiveGateVersionResponse) IsEmpty() bool {
	return len(lr.LatestGatewayVersion) == 0
}

// GetLatestActiveGateVersion gets the latest gateway version for the given OS and arch configured on the Tenant.
func (c *ClientImpl) GetLatestActiveGateVersion(ctx context.Context, os string) (string, error) {
	if len(os) == 0 {
		return "", errEmptyOS
	}

	var resp latestActiveGateVersionResponse

	err := c.apiClient.GET(ctx, "/v1/deployment/installer/gateway").
		WithPath(os, "latest/metainfo").
		WithPaasToken().Execute(&resp)

	return resp.LatestGatewayVersion, errors.WithStack(err)
}

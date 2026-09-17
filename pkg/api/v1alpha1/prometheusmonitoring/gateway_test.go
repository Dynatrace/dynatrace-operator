// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package prometheusmonitoring

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGateway(t *testing.T) {
	spec := &GatewaySpec{}

	gateway := NewGateway(spec, "pm")

	// The wrapper aliases the spec rather than copying it, so a change through the accessor is
	// visible on the PrometheusMonitoring it came from.
	assert.Same(t, spec, gateway.GatewaySpec)
	assert.Equal(t, "pm-gateway", gateway.GetStatefulSetName())
}

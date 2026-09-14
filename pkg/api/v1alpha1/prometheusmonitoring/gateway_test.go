// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package prometheusmonitoring

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewGateway(t *testing.T) {
	spec := &GatewaySpec{}

	gateway := NewGateway(spec, "pm")

	assert.Same(t, spec, gateway.GatewaySpec)
	assert.Equal(t, "pm"+GatewayNameSuffix, gateway.GetStatefulSetName())
}

func TestGateway_GetStatefulSetName(t *testing.T) {
	gateway := NewGateway(&GatewaySpec{}, "pm")

	assert.Equal(t, "pm-gateway", gateway.GetStatefulSetName())
}

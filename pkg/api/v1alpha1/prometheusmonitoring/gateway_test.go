// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package prometheusmonitoring

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewGateway(t *testing.T) {
	spec := &GatewaySpec{}

	gateway := NewGateway(spec, "dtp")

	assert.Same(t, spec, gateway.GatewaySpec)
	assert.Equal(t, "dtp"+GatewayNameSuffix, gateway.GetStatefulSetName())
}

func TestGateway_GetStatefulSetName(t *testing.T) {
	gateway := NewGateway(&GatewaySpec{}, "dtp")

	assert.Equal(t, "dtp-gateway", gateway.GetStatefulSetName())
}

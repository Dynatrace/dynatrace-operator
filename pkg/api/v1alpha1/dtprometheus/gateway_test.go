// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package dtprometheus

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewGateway(t *testing.T) {
	spec := &GatewaySpec{}

	gateway := NewGateway(spec, "dtprom")

	assert.Same(t, spec, gateway.GatewaySpec)
	assert.Equal(t, "dtprom"+GatewayNameSuffix, gateway.GetStatefulSetName())
}

func TestGateway_GetStatefulSetName(t *testing.T) {
	gateway := NewGateway(&GatewaySpec{}, "dtprom")

	assert.Equal(t, "dtprom-gateway", gateway.GetStatefulSetName())
}

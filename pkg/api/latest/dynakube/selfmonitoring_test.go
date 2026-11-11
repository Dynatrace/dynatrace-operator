// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package dynakube

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSelfMonitoringOTLPEndpoint(t *testing.T) {
	dk := &DynaKube{Name: "my-dk"}

	assert.Equal(t, "my-dk-self-monitoring", dk.SelfMonitoringName())
	assert.Equal(t, "http://my-dk-self-monitoring:4318", dk.SelfMonitoringOTLPEndpoint())
}

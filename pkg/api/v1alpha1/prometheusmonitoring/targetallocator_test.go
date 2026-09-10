// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package prometheusmonitoring

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewTargetAllocator(t *testing.T) {
	spec := &TargetAllocatorSpec{}

	ta := NewTargetAllocator(spec, "dtp")

	assert.Same(t, spec, ta.TargetAllocatorSpec)
	assert.Equal(t, "dtp"+TargetAllocatorNameSuffix, ta.GetDeploymentName())
}

func TestTargetAllocator_GetDeploymentName(t *testing.T) {
	ta := NewTargetAllocator(&TargetAllocatorSpec{}, "dtp")

	assert.Equal(t, "dtp-allocator", ta.GetDeploymentName())
}

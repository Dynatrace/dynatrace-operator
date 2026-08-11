// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package dtprometheus

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewTargetAllocator(t *testing.T) {
	spec := &TargetAllocatorSpec{}

	ta := NewTargetAllocator(spec, "dtprom")

	assert.Same(t, spec, ta.TargetAllocatorSpec)
	assert.Equal(t, "dtprom"+TargetAllocatorNameSuffix, ta.GetDeploymentName())
}

func TestTargetAllocator_GetDeploymentName(t *testing.T) {
	ta := NewTargetAllocator(&TargetAllocatorSpec{}, "dtprom")

	assert.Equal(t, "dtprom-target-allocator", ta.GetDeploymentName())
}

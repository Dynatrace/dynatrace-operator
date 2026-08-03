// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package dtprometheus

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func TestTargetAllocator_SetName(t *testing.T) {
	ta := NewTargetAllocator(&TargetAllocatorSpec{}, "old")

	ta.SetName("new")

	assert.Equal(t, "new-target-allocator", ta.GetDeploymentName())
}

func TestTargetAllocator_GetScrapeCRSelector(t *testing.T) {
	t.Run("unset returns default label selector", func(t *testing.T) {
		ta := NewTargetAllocator(&TargetAllocatorSpec{}, "dtprom")

		selector := ta.GetScrapeCRSelector()

		require.NotNil(t, selector)
		assert.Equal(t, map[string]string{DefaultScrapeCRSelectorLabel: "true"}, selector.MatchLabels)
	})

	t.Run("set returns configured selector", func(t *testing.T) {
		configured := &metav1.LabelSelector{MatchLabels: map[string]string{"custom": "value"}}
		ta := NewTargetAllocator(&TargetAllocatorSpec{ScrapeCRSelector: configured}, "dtprom")

		assert.Same(t, configured, ta.GetScrapeCRSelector())
	})

	t.Run("empty selector matches all CRDs", func(t *testing.T) {
		empty := &metav1.LabelSelector{}
		ta := NewTargetAllocator(&TargetAllocatorSpec{ScrapeCRSelector: empty}, "dtprom")

		assert.Same(t, empty, ta.GetScrapeCRSelector())
	})
}

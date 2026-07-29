// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package targetallocator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestNew(t *testing.T) {
	spec := &Spec{}

	ta := New(spec, "dtprom")

	assert.Same(t, spec, ta.Spec)
	assert.Equal(t, "dtprom"+NameSuffix, ta.GetName())
}

func TestTargetAllocator_GetName(t *testing.T) {
	ta := New(&Spec{}, "dtprom")

	assert.Equal(t, "dtprom-target-allocator", ta.GetName())
}

func TestTargetAllocator_SetName(t *testing.T) {
	ta := New(&Spec{}, "old")

	ta.SetName("new")

	assert.Equal(t, "new-target-allocator", ta.GetName())
}

func TestTargetAllocator_GetScrapeInterval(t *testing.T) {
	tests := []struct {
		name     string
		interval string
		expected string
	}{
		{"unset returns default", "", DefaultScrapeInterval},
		{"set returns value", "30s", "30s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ta := New(&Spec{ScrapeInterval: tt.interval}, "dtprom")
			assert.Equal(t, tt.expected, ta.GetScrapeInterval())
		})
	}
}

func TestTargetAllocator_GetScrapeCRSelector(t *testing.T) {
	t.Run("unset returns default label selector", func(t *testing.T) {
		ta := New(&Spec{}, "dtprom")

		selector := ta.GetScrapeCRSelector()

		require.NotNil(t, selector)
		assert.Equal(t, map[string]string{DefaultScrapeCRSelectorLabel: "true"}, selector.MatchLabels)
	})

	t.Run("set returns configured selector", func(t *testing.T) {
		configured := &metav1.LabelSelector{MatchLabels: map[string]string{"custom": "value"}}
		ta := New(&Spec{ScrapeCRSelector: configured}, "dtprom")

		assert.Same(t, configured, ta.GetScrapeCRSelector())
	})

	t.Run("empty selector matches all CRDs", func(t *testing.T) {
		empty := &metav1.LabelSelector{}
		ta := New(&Spec{ScrapeCRSelector: empty}, "dtprom")

		assert.Same(t, empty, ta.GetScrapeCRSelector())
	})
}

func TestTargetAllocator_GetScrapeCRNamespaceSelector(t *testing.T) {
	t.Run("unset returns nil", func(t *testing.T) {
		ta := New(&Spec{}, "dtprom")

		assert.Nil(t, ta.GetScrapeCRNamespaceSelector())
	})

	t.Run("set returns configured selector", func(t *testing.T) {
		configured := &metav1.LabelSelector{MatchLabels: map[string]string{"team": "monitoring"}}
		ta := New(&Spec{ScrapeCRNamespaceSelector: configured}, "dtprom")

		assert.Same(t, configured, ta.GetScrapeCRNamespaceSelector())
	})
}

func TestTargetAllocator_GetUpdateStrategy(t *testing.T) {
	strategy := appsv1.DeploymentStrategy{
		Type: appsv1.RollingUpdateDeploymentStrategyType,
		RollingUpdate: &appsv1.RollingUpdateDeployment{
			MaxUnavailable: new(intstr.FromInt32(1)),
		},
	}

	ta := New(&Spec{UpdateStrategy: strategy}, "dtprom")

	assert.Equal(t, strategy, ta.GetUpdateStrategy())
}

// TestTargetAllocator_PromotesCommonGetters verifies that the shared common.Spec
// getters are promoted through the wrapper's embedded *Spec.
func TestTargetAllocator_PromotesCommonGetters(t *testing.T) {
	ta := New(&Spec{}, "dtprom")
	assert.Equal(t, int32(2), ta.GetReplicas())

	ta.Replicas = new(int32(3))
	assert.Equal(t, int32(3), ta.GetReplicas())
}

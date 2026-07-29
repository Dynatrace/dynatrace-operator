// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package selfmonitoring

import (
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestNew(t *testing.T) {
	spec := &Spec{}

	selfMonitoring := New(spec, "dtprom")

	assert.Same(t, spec, selfMonitoring.Spec)
	assert.Equal(t, "dtprom"+NameSuffix, selfMonitoring.GetName())
}

func TestSelfMonitoring_GetName(t *testing.T) {
	selfMonitoring := New(&Spec{}, "dtprom")

	assert.Equal(t, "dtprom-self-monitoring", selfMonitoring.GetName())
}

func TestSelfMonitoring_SetName(t *testing.T) {
	selfMonitoring := New(&Spec{}, "old")

	selfMonitoring.SetName("new")

	assert.Equal(t, "new-self-monitoring", selfMonitoring.GetName())
}

func TestSelfMonitoring_IsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		enabled  bool
		expected bool
	}{
		{"disabled by default", false, false},
		{"enabled", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selfMonitoring := New(&Spec{Enabled: tt.enabled}, "dtprom")
			assert.Equal(t, tt.expected, selfMonitoring.IsEnabled())
		})
	}
}

func TestSelfMonitoring_GetUpdateStrategy(t *testing.T) {
	strategy := appsv1.DeploymentStrategy{
		Type: appsv1.RollingUpdateDeploymentStrategyType,
		RollingUpdate: &appsv1.RollingUpdateDeployment{
			MaxUnavailable: new(intstr.FromInt32(1)),
		},
	}

	selfMonitoring := New(&Spec{UpdateStrategy: strategy}, "dtprom")

	assert.Equal(t, strategy, selfMonitoring.GetUpdateStrategy())
}

// TestSelfMonitoring_PromotesCommonGetters verifies that the shared common.Spec
// getters are promoted through the wrapper's embedded *Spec.
func TestSelfMonitoring_PromotesCommonGetters(t *testing.T) {
	selfMonitoring := New(&Spec{}, "dtprom")
	assert.Equal(t, int32(2), selfMonitoring.GetReplicas())

	selfMonitoring.Replicas = new(int32(3))
	assert.Equal(t, int32(3), selfMonitoring.GetReplicas())
}

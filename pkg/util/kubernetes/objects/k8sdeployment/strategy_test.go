// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package k8sdeployment

import (
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestNormalizeStrategy(t *testing.T) {
	percent25 := intstr.FromString("25%")
	percent10 := intstr.FromString("10%")
	one := intstr.FromInt(1)

	tests := []struct {
		name    string
		desired appsv1.DeploymentStrategy
		want    appsv1.DeploymentStrategy
	}{
		{
			name:    "empty strategy defaults to RollingUpdate with 25%/25%",
			desired: appsv1.DeploymentStrategy{},
			want: appsv1.DeploymentStrategy{
				Type:          appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{MaxUnavailable: &percent25, MaxSurge: &percent25},
			},
		},
		{
			name:    "type only fills in rollingUpdate defaults",
			desired: appsv1.DeploymentStrategy{Type: appsv1.RollingUpdateDeploymentStrategyType},
			want: appsv1.DeploymentStrategy{
				Type:          appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{MaxUnavailable: &percent25, MaxSurge: &percent25},
			},
		},
		{
			name:    "rollingUpdate only, no type, is treated as RollingUpdate",
			desired: appsv1.DeploymentStrategy{RollingUpdate: &appsv1.RollingUpdateDeployment{}},
			want: appsv1.DeploymentStrategy{
				Type:          appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{MaxUnavailable: &percent25, MaxSurge: &percent25},
			},
		},
		{
			name: "Recreate clears a stale rollingUpdate block",
			desired: appsv1.DeploymentStrategy{
				Type:          appsv1.RecreateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{MaxUnavailable: &percent10},
			},
			want: appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType},
		},
		{
			name:    "non-RollingUpdate type clears a stale rollingUpdate block",
			desired: appsv1.DeploymentStrategy{Type: appsv1.DeploymentStrategyType("Custom"), RollingUpdate: &appsv1.RollingUpdateDeployment{MaxSurge: &one}},
			want:    appsv1.DeploymentStrategy{Type: appsv1.DeploymentStrategyType("Custom")},
		},
		{
			name: "user-provided values are kept",
			desired: appsv1.DeploymentStrategy{
				Type:          appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{MaxUnavailable: &percent10, MaxSurge: &one},
			},
			want: appsv1.DeploymentStrategy{
				Type:          appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{MaxUnavailable: &percent10, MaxSurge: &one},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeStrategy(tt.desired)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNormalizeStrategyDoesNotMutateInput(t *testing.T) {
	desired := appsv1.DeploymentStrategy{RollingUpdate: &appsv1.RollingUpdateDeployment{}}
	original := *desired.DeepCopy()

	NormalizeStrategy(desired)

	assert.Equal(t, original, desired)
}

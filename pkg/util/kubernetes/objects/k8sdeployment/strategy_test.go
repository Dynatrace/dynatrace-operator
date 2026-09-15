// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package k8sdeployment

import (
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestMergeStrategy(t *testing.T) {
	// defaulted is what the apiserver stores for a RollingUpdate deployment when nothing is set.
	defaulted := func() appsv1.DeploymentStrategy {
		return appsv1.DeploymentStrategy{
			Type: appsv1.RollingUpdateDeploymentStrategyType,
			RollingUpdate: &appsv1.RollingUpdateDeployment{
				MaxUnavailable: new(intstr.FromString("25%")),
				MaxSurge:       new(intstr.FromString("25%")),
			},
		}
	}

	tests := []struct {
		name    string
		current appsv1.DeploymentStrategy
		desired appsv1.DeploymentStrategy
		want    appsv1.DeploymentStrategy
	}{
		{
			name:    "nothing set keeps the apiserver defaults on the stored object",
			current: defaulted(),
			desired: appsv1.DeploymentStrategy{},
			want:    defaulted(),
		},
		{
			name:    "nothing set on a new object stays empty, the apiserver defaults it on create",
			current: appsv1.DeploymentStrategy{},
			desired: appsv1.DeploymentStrategy{},
			want:    appsv1.DeploymentStrategy{},
		},
		{
			name:    "type only overrides the type and keeps the stored rollingUpdate",
			current: defaulted(),
			desired: appsv1.DeploymentStrategy{Type: appsv1.RollingUpdateDeploymentStrategyType},
			want:    defaulted(),
		},
		{
			name:    "partial rollingUpdate keeps the stored value of the other field",
			current: defaulted(),
			desired: appsv1.DeploymentStrategy{RollingUpdate: &appsv1.RollingUpdateDeployment{MaxUnavailable: new(intstr.FromInt32(0))}},
			want: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxUnavailable: new(intstr.FromInt32(0)),
					MaxSurge:       new(intstr.FromString("25%")),
				},
			},
		},
		{
			name:    "rollingUpdate on a new object only carries what the spec sets",
			current: appsv1.DeploymentStrategy{},
			desired: appsv1.DeploymentStrategy{RollingUpdate: &appsv1.RollingUpdateDeployment{MaxUnavailable: new(intstr.FromInt32(0))}},
			want:    appsv1.DeploymentStrategy{RollingUpdate: &appsv1.RollingUpdateDeployment{MaxUnavailable: new(intstr.FromInt32(0))}},
		},
		{
			name:    "switching to Recreate clears the stored rollingUpdate block",
			current: defaulted(),
			desired: appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType},
			want:    appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType},
		},
		{
			name:    "a stored Recreate is kept when the spec no longer sets a type",
			current: appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType},
			desired: appsv1.DeploymentStrategy{},
			want:    appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType},
		},
		{
			name:    "fully specified values win over the stored ones",
			current: defaulted(),
			desired: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxUnavailable: new(intstr.FromString("10%")),
					MaxSurge:       new(intstr.FromInt32(1)),
				},
			},
			want: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxUnavailable: new(intstr.FromString("10%")),
					MaxSurge:       new(intstr.FromInt32(1)),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MergeStrategy(tt.current, tt.desired)

			assert.Equal(t, tt.want, got)
		})
	}
}

// The merge runs on the stored object, so writing through its rollingUpdate pointer would change the
// object the caller still compares against.
func TestMergeStrategyDoesNotMutateInputs(t *testing.T) {
	current := appsv1.DeploymentStrategy{
		Type: appsv1.RollingUpdateDeploymentStrategyType,
		RollingUpdate: &appsv1.RollingUpdateDeployment{
			MaxUnavailable: new(intstr.FromString("25%")),
			MaxSurge:       new(intstr.FromString("25%")),
		},
	}
	desired := appsv1.DeploymentStrategy{
		RollingUpdate: &appsv1.RollingUpdateDeployment{MaxUnavailable: new(intstr.FromInt32(0))},
	}

	currentBefore := *current.DeepCopy()
	desiredBefore := *desired.DeepCopy()

	MergeStrategy(current, desired)

	assert.Equal(t, currentBefore, current)
	assert.Equal(t, desiredBefore, desired)
}

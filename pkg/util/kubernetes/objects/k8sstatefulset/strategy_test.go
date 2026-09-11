// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package k8sstatefulset

import (
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestNormalizeUpdateStrategy(t *testing.T) {
	zero := int32(0)
	five := int32(5)
	one := intstr.FromInt(1)

	tests := []struct {
		name    string
		current appsv1.StatefulSetUpdateStrategy
		desired appsv1.StatefulSetUpdateStrategy
		want    appsv1.StatefulSetUpdateStrategy
	}{
		{
			name:    "empty strategy defaults to RollingUpdate with partition 0",
			desired: appsv1.StatefulSetUpdateStrategy{},
			want: appsv1.StatefulSetUpdateStrategy{
				Type:          appsv1.RollingUpdateStatefulSetStrategyType,
				RollingUpdate: &appsv1.RollingUpdateStatefulSetStrategy{Partition: &zero},
			},
		},
		{
			name:    "type only fills in rollingUpdate defaults",
			desired: appsv1.StatefulSetUpdateStrategy{Type: appsv1.RollingUpdateStatefulSetStrategyType},
			want: appsv1.StatefulSetUpdateStrategy{
				Type:          appsv1.RollingUpdateStatefulSetStrategyType,
				RollingUpdate: &appsv1.RollingUpdateStatefulSetStrategy{Partition: &zero},
			},
		},
		{
			name:    "rollingUpdate only, no type, is treated as RollingUpdate",
			desired: appsv1.StatefulSetUpdateStrategy{RollingUpdate: &appsv1.RollingUpdateStatefulSetStrategy{}},
			want: appsv1.StatefulSetUpdateStrategy{
				Type:          appsv1.RollingUpdateStatefulSetStrategyType,
				RollingUpdate: &appsv1.RollingUpdateStatefulSetStrategy{Partition: &zero},
			},
		},
		{
			name: "OnDelete clears a stale rollingUpdate block",
			desired: appsv1.StatefulSetUpdateStrategy{
				Type:          appsv1.OnDeleteStatefulSetStrategyType,
				RollingUpdate: &appsv1.RollingUpdateStatefulSetStrategy{Partition: &five},
			},
			want: appsv1.StatefulSetUpdateStrategy{Type: appsv1.OnDeleteStatefulSetStrategyType},
		},
		{
			name: "user-provided values are kept",
			desired: appsv1.StatefulSetUpdateStrategy{
				Type:          appsv1.RollingUpdateStatefulSetStrategyType,
				RollingUpdate: &appsv1.RollingUpdateStatefulSetStrategy{Partition: &five, MaxUnavailable: &one},
			},
			want: appsv1.StatefulSetUpdateStrategy{
				Type:          appsv1.RollingUpdateStatefulSetStrategyType,
				RollingUpdate: &appsv1.RollingUpdateStatefulSetStrategy{Partition: &five, MaxUnavailable: &one},
			},
		},
		{
			name:    "nil maxUnavailable with no current rollingUpdate stays nil",
			current: appsv1.StatefulSetUpdateStrategy{},
			desired: appsv1.StatefulSetUpdateStrategy{RollingUpdate: &appsv1.RollingUpdateStatefulSetStrategy{Partition: &five}},
			want: appsv1.StatefulSetUpdateStrategy{
				Type:          appsv1.RollingUpdateStatefulSetStrategyType,
				RollingUpdate: &appsv1.RollingUpdateStatefulSetStrategy{Partition: &five},
			},
		},
		{
			name: "nil maxUnavailable is taken from the current, apiserver-defaulted object",
			current: appsv1.StatefulSetUpdateStrategy{
				Type:          appsv1.RollingUpdateStatefulSetStrategyType,
				RollingUpdate: &appsv1.RollingUpdateStatefulSetStrategy{Partition: &zero, MaxUnavailable: &one},
			},
			desired: appsv1.StatefulSetUpdateStrategy{RollingUpdate: &appsv1.RollingUpdateStatefulSetStrategy{Partition: &five}},
			want: appsv1.StatefulSetUpdateStrategy{
				Type:          appsv1.RollingUpdateStatefulSetStrategyType,
				RollingUpdate: &appsv1.RollingUpdateStatefulSetStrategy{Partition: &five, MaxUnavailable: &one},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeUpdateStrategy(tt.current, tt.desired)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNormalizeUpdateStrategyDoesNotMutateInput(t *testing.T) {
	one := intstr.FromInt(1)
	current := appsv1.StatefulSetUpdateStrategy{
		Type:          appsv1.RollingUpdateStatefulSetStrategyType,
		RollingUpdate: &appsv1.RollingUpdateStatefulSetStrategy{MaxUnavailable: &one},
	}
	originalCurrent := *current.DeepCopy()
	desired := appsv1.StatefulSetUpdateStrategy{RollingUpdate: &appsv1.RollingUpdateStatefulSetStrategy{}}
	originalDesired := *desired.DeepCopy()

	NormalizeUpdateStrategy(current, desired)

	assert.Equal(t, originalCurrent, current)
	assert.Equal(t, originalDesired, desired)
}

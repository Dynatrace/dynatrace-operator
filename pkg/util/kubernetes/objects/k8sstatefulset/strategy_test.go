// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package k8sstatefulset

import (
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestMergeUpdateStrategy(t *testing.T) {
	// defaulted is what the apiserver stores for a RollingUpdate statefulset when nothing is set.
	defaulted := func() appsv1.StatefulSetUpdateStrategy {
		return appsv1.StatefulSetUpdateStrategy{
			Type:          appsv1.RollingUpdateStatefulSetStrategyType,
			RollingUpdate: &appsv1.RollingUpdateStatefulSetStrategy{Partition: new(int32(0))},
		}
	}

	tests := []struct {
		name    string
		current appsv1.StatefulSetUpdateStrategy
		desired appsv1.StatefulSetUpdateStrategy
		want    appsv1.StatefulSetUpdateStrategy
	}{
		{
			name:    "nothing set keeps the apiserver defaults on the stored object",
			current: defaulted(),
			desired: appsv1.StatefulSetUpdateStrategy{},
			want:    defaulted(),
		},
		{
			name:    "nothing set on a new object stays empty, the apiserver defaults it on create",
			current: appsv1.StatefulSetUpdateStrategy{},
			desired: appsv1.StatefulSetUpdateStrategy{},
			want:    appsv1.StatefulSetUpdateStrategy{},
		},
		{
			name:    "type only overrides the type and keeps the stored rollingUpdate",
			current: defaulted(),
			desired: appsv1.StatefulSetUpdateStrategy{Type: appsv1.RollingUpdateStatefulSetStrategyType},
			want:    defaulted(),
		},
		{
			name: "partition overrides the stored one and keeps the stored maxUnavailable",
			current: appsv1.StatefulSetUpdateStrategy{
				Type:          appsv1.RollingUpdateStatefulSetStrategyType,
				RollingUpdate: &appsv1.RollingUpdateStatefulSetStrategy{Partition: new(int32(0)), MaxUnavailable: new(intstr.FromInt32(1))},
			},
			desired: appsv1.StatefulSetUpdateStrategy{
				RollingUpdate: &appsv1.RollingUpdateStatefulSetStrategy{Partition: new(int32(2))},
			},
			want: appsv1.StatefulSetUpdateStrategy{
				Type:          appsv1.RollingUpdateStatefulSetStrategyType,
				RollingUpdate: &appsv1.RollingUpdateStatefulSetStrategy{Partition: new(int32(2)), MaxUnavailable: new(intstr.FromInt32(1))},
			},
		},
		{
			name:    "rollingUpdate on a new object only carries what the spec sets",
			current: appsv1.StatefulSetUpdateStrategy{},
			desired: appsv1.StatefulSetUpdateStrategy{
				RollingUpdate: &appsv1.RollingUpdateStatefulSetStrategy{Partition: new(int32(1))},
			},
			want: appsv1.StatefulSetUpdateStrategy{
				RollingUpdate: &appsv1.RollingUpdateStatefulSetStrategy{Partition: new(int32(1))},
			},
		},
		{
			name:    "switching to OnDelete clears the stored rollingUpdate block",
			current: defaulted(),
			desired: appsv1.StatefulSetUpdateStrategy{Type: appsv1.OnDeleteStatefulSetStrategyType},
			want:    appsv1.StatefulSetUpdateStrategy{Type: appsv1.OnDeleteStatefulSetStrategyType},
		},
		{
			name:    "a stored OnDelete is kept when the spec no longer sets a type",
			current: appsv1.StatefulSetUpdateStrategy{Type: appsv1.OnDeleteStatefulSetStrategyType},
			desired: appsv1.StatefulSetUpdateStrategy{},
			want:    appsv1.StatefulSetUpdateStrategy{Type: appsv1.OnDeleteStatefulSetStrategyType},
		},
		{
			name:    "fully specified values win over the stored ones",
			current: defaulted(),
			desired: appsv1.StatefulSetUpdateStrategy{
				Type:          appsv1.RollingUpdateStatefulSetStrategyType,
				RollingUpdate: &appsv1.RollingUpdateStatefulSetStrategy{Partition: new(int32(3)), MaxUnavailable: new(intstr.FromInt32(1))},
			},
			want: appsv1.StatefulSetUpdateStrategy{
				Type:          appsv1.RollingUpdateStatefulSetStrategyType,
				RollingUpdate: &appsv1.RollingUpdateStatefulSetStrategy{Partition: new(int32(3)), MaxUnavailable: new(intstr.FromInt32(1))},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MergeUpdateStrategy(tt.current, tt.desired)

			assert.Equal(t, tt.want, got)
		})
	}
}

// The merge runs on the stored object, so writing through its rollingUpdate pointer would change the
// object the caller still compares against.
func TestMergeUpdateStrategyDoesNotMutateInputs(t *testing.T) {
	current := appsv1.StatefulSetUpdateStrategy{
		Type:          appsv1.RollingUpdateStatefulSetStrategyType,
		RollingUpdate: &appsv1.RollingUpdateStatefulSetStrategy{Partition: new(int32(0))},
	}
	desired := appsv1.StatefulSetUpdateStrategy{
		RollingUpdate: &appsv1.RollingUpdateStatefulSetStrategy{Partition: new(int32(2))},
	}

	currentBefore := *current.DeepCopy()
	desiredBefore := *desired.DeepCopy()

	MergeUpdateStrategy(current, desired)

	assert.Equal(t, currentBefore, current)
	assert.Equal(t, desiredBefore, desired)
}

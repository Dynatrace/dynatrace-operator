// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package dtprometheus

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/image"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestPodSpec_GetPullPolicy(t *testing.T) {
	tests := []struct {
		name     string
		policy   image.PullPolicy
		expected corev1.PullPolicy
	}{
		{"unset", "", corev1.PullPolicy("")},
		{"always", image.PullPolicy("Always"), corev1.PullAlways},
		{"if not present", image.PullPolicy("IfNotPresent"), corev1.PullIfNotPresent},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := &PodSpec{ImagePullPolicy: tt.policy}
			assert.Equal(t, tt.expected, spec.GetPullPolicy())
		})
	}
}

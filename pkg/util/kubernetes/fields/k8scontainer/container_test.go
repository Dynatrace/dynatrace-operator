// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package k8scontainer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

const (
	containerName = "test-name"
	doNotFindName = "do-not-find"
	podName       = "test-pod"
)

var podSpec = corev1.PodSpec{
	Containers: []corev1.Container{
		{
			Name: containerName,
		},
	},
}

func TestFindInPodSpec(t *testing.T) {
	t.Run("container is found in podSpec", func(t *testing.T) {
		containerInPodSpec := FindInPodSpec(&podSpec, containerName)
		require.NotNil(t, containerInPodSpec)
	})
	t.Run("container is not found in podSpec", func(t *testing.T) {
		containerInPodSpec := FindInPodSpec(&podSpec, doNotFindName)
		require.Nil(t, containerInPodSpec)
	})
}

func TestGetFirstInPodSpec(t *testing.T) {
	t.Run("returns the first container", func(t *testing.T) {
		spec := corev1.PodSpec{
			Containers: []corev1.Container{{Name: "first"}, {Name: "second"}},
		}

		assert.Equal(t, "first", GetFirstInPodSpec(&spec).Name)
	})
	t.Run("returns the zero container when there are none", func(t *testing.T) {
		assert.Equal(t, corev1.Container{}, GetFirstInPodSpec(&corev1.PodSpec{}))
	})
}

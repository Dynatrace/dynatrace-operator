// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package mutator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewContainers(t *testing.T) {
	neverInjected := func(_ corev1.Container, _ *BaseRequest) bool { return false }

	t.Run("placeholder container is excluded", func(t *testing.T) {
		req := &BaseRequest{
			Pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "app", Image: "my-app:latest"},
						{Name: "istio-placeholder", Image: plaholderContainerImageName},
					},
				},
			},
		}

		containers := req.NewContainers(neverInjected)

		assert.Len(t, containers, 1)
		assert.Equal(t, "app", containers[0].Name)
	})

	t.Run("all placeholder containers are excluded", func(t *testing.T) {
		req := &BaseRequest{
			Pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "placeholder-1", Image: plaholderContainerImageName},
						{Name: "placeholder-2", Image: plaholderContainerImageName},
					},
				},
			},
		}

		containers := req.NewContainers(neverInjected)

		assert.Empty(t, containers)
	})

	t.Run("non-placeholder containers are included", func(t *testing.T) {
		req := &BaseRequest{
			Pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "app", Image: "my-app:latest"},
						{Name: "sidecar", Image: "sidecar:v1"},
					},
				},
			},
		}

		containers := req.NewContainers(neverInjected)

		assert.Len(t, containers, 2)
	})

	t.Run("already injected containers are excluded", func(t *testing.T) {
		alwaysInjected := func(_ corev1.Container, _ *BaseRequest) bool { return true }

		req := &BaseRequest{
			Pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "app", Image: "my-app:latest"},
					},
				},
			},
		}

		containers := req.NewContainers(alwaysInjected)

		assert.Empty(t, containers)
	})

	t.Run("annotation-excluded containers are excluded", func(t *testing.T) {
		req := &BaseRequest{
			Pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"container.inject.dynatrace.com/app": "false",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "app", Image: "my-app:latest"},
						{Name: "other", Image: "other:latest"},
					},
				},
			},
		}

		containers := req.NewContainers(neverInjected)

		assert.Len(t, containers, 1)
		assert.Equal(t, "other", containers[0].Name)
	})
}

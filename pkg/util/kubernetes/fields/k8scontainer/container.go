// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package k8scontainer

import (
	corev1 "k8s.io/api/core/v1"
)

func FindInPodSpec(podSpec *corev1.PodSpec, containerName string) *corev1.Container {
	for i := range podSpec.Containers {
		container := &podSpec.Containers[i]
		if container.Name == containerName {
			return container
		}
	}

	return nil
}

// GetFirstInPodSpec returns the first container of the given pod spec, or the zero
// Container when the pod spec has none. Useful when reconciling a workload whose
// pod template holds exactly one container and the stored, apiserver-defaulted
// container is needed to avoid spurious diffs.
func GetFirstInPodSpec(podSpec *corev1.PodSpec) corev1.Container {
	if len(podSpec.Containers) > 0 {
		return podSpec.Containers[0]
	}

	return corev1.Container{}
}

func FindInitInPodSpec(podSpec *corev1.PodSpec, containerName string) *corev1.Container {
	for i := range podSpec.InitContainers {
		container := &podSpec.InitContainers[i]
		if container.Name == containerName {
			return container
		}
	}

	return nil
}

// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package types

import (
	corev1 "k8s.io/api/core/v1"
)

// +kubebuilder:object:generate=true

type Volume struct {
	Name                string `json:"name"`
	AllowedVolumeSource `json:",inline"`
}

func (v Volume) AsCoreV1() corev1.Volume {
	return corev1.Volume{
		Name:         v.Name,
		VolumeSource: v.AllowedVolumeSource.AsCoreV1(),
	}
}

// +kubebuilder:object:generate=true

type Volumes []Volume

func (v Volumes) AsCoreV1() []corev1.Volume {
	volumes := make([]corev1.Volume, len(v))
	for i, volume := range v {
		volumes[i] = volume.AsCoreV1()
	}

	return volumes
}

// +kubebuilder:object:generate=true

type AllowedVolumeSource struct {
	ConfigMap             *corev1.ConfigMapVolumeSource             `json:"configMap,omitempty"`
	CSI                   *corev1.CSIVolumeSource                   `json:"csi,omitempty"`
	DownwardAPI           *corev1.DownwardAPIVolumeSource           `json:"downwardAPI,omitempty"`
	EmptyDir              *corev1.EmptyDirVolumeSource              `json:"emptyDir,omitempty"`
	Ephemeral             *corev1.EphemeralVolumeSource             `json:"ephemeral,omitempty"`
	PersistentVolumeClaim *corev1.PersistentVolumeClaimVolumeSource `json:"persistentVolumeClaim,omitempty"`
	Projected             *corev1.ProjectedVolumeSource             `json:"projected,omitempty"`
	Secret                *corev1.SecretVolumeSource                `json:"secret,omitempty"`
	Image                 *corev1.ImageVolumeSource                 `json:"image,omitempty"`
}

func (vs AllowedVolumeSource) AsCoreV1() corev1.VolumeSource {
	return corev1.VolumeSource{
		ConfigMap:             vs.ConfigMap,
		CSI:                   vs.CSI,
		DownwardAPI:           vs.DownwardAPI,
		EmptyDir:              vs.EmptyDir,
		Ephemeral:             vs.Ephemeral,
		PersistentVolumeClaim: vs.PersistentVolumeClaim,
		Projected:             vs.Projected,
		Secret:                vs.Secret,
		Image:                 vs.Image,
	}
}

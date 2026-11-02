// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package volume

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestAllowedSourceAsCoreV1(t *testing.T) {
	t.Run("empty source maps to empty volume source", func(t *testing.T) {
		assert.Equal(t, corev1.VolumeSource{}, AllowedSource{}.AsCoreV1())
	})

	tests := []struct {
		name     string
		source   AllowedSource
		expected corev1.VolumeSource
	}{
		{"configMap", AllowedSource{ConfigMap: &corev1.ConfigMapVolumeSource{}}, corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{}}},
		{"csi", AllowedSource{CSI: &corev1.CSIVolumeSource{Driver: "d"}}, corev1.VolumeSource{CSI: &corev1.CSIVolumeSource{Driver: "d"}}},
		{"downwardAPI", AllowedSource{DownwardAPI: &corev1.DownwardAPIVolumeSource{}}, corev1.VolumeSource{DownwardAPI: &corev1.DownwardAPIVolumeSource{}}},
		{"emptyDir", AllowedSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}, corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
		{"ephemeral", AllowedSource{Ephemeral: &corev1.EphemeralVolumeSource{}}, corev1.VolumeSource{Ephemeral: &corev1.EphemeralVolumeSource{}}},
		{"persistentVolumeClaim", AllowedSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "c"}}, corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "c"}}},
		{"projected", AllowedSource{Projected: &corev1.ProjectedVolumeSource{}}, corev1.VolumeSource{Projected: &corev1.ProjectedVolumeSource{}}},
		{"secret", AllowedSource{Secret: &corev1.SecretVolumeSource{SecretName: "s"}}, corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: "s"}}},
		{"image", AllowedSource{Image: &corev1.ImageVolumeSource{Reference: "r"}}, corev1.VolumeSource{Image: &corev1.ImageVolumeSource{Reference: "r"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.source.AsCoreV1())
		})
	}

	t.Run("pointers are shared, not copied", func(t *testing.T) {
		emptyDir := &corev1.EmptyDirVolumeSource{}

		assert.Same(t, emptyDir, AllowedSource{EmptyDir: emptyDir}.AsCoreV1().EmptyDir)
	})
}

func TestSpecAsCoreV1(t *testing.T) {
	spec := Spec{
		Name:   "my-volume",
		Secret: &corev1.SecretVolumeSource{SecretName: "s"},
	}

	assert.Equal(t, corev1.Volume{
		Name:   "my-volume",
		Secret: &corev1.SecretVolumeSource{SecretName: "s"},
	}, spec.AsCoreV1())
}

func TestSpecsAsCoreV1(t *testing.T) {
	t.Run("nil specs result in empty slice", func(t *testing.T) {
		assert.Empty(t, Specs(nil).AsCoreV1())
	})

	t.Run("order and content are preserved", func(t *testing.T) {
		specs := Specs{
			{Name: "a", EmptyDir: &corev1.EmptyDirVolumeSource{}},
			{Name: "b", ConfigMap: &corev1.ConfigMapVolumeSource{}},
		}

		assert.Equal(t, []corev1.Volume{
			{Name: "a", EmptyDir: &corev1.EmptyDirVolumeSource{}},
			{Name: "b", ConfigMap: &corev1.ConfigMapVolumeSource{}},
		}, specs.AsCoreV1())
	})
}

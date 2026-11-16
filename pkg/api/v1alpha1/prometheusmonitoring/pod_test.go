// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package prometheusmonitoring

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/types"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestAppendEnv(t *testing.T) {
	base := []corev1.EnvVar{{Name: "A", Value: "base"}}

	t.Run("empty", func(t *testing.T) {
		assert.Equal(t, base, PodSpec{}.AppendEnv(base))
	})

	t.Run("appends after base", func(t *testing.T) {
		spec := PodSpec{Env: []corev1.EnvVar{{Name: "B", Value: "custom"}}}

		assert.Equal(t, []corev1.EnvVar{{Name: "A", Value: "base"}, {Name: "B", Value: "custom"}}, spec.AppendEnv(base))
	})

	t.Run("duplicate name keeps the custom entry last", func(t *testing.T) {
		spec := PodSpec{Env: []corev1.EnvVar{{Name: "A", Value: "custom"}}}

		assert.Equal(t, []corev1.EnvVar{{Name: "A", Value: "base"}, {Name: "A", Value: "custom"}}, spec.AppendEnv(base))
	})

	t.Run("does not modify base", func(t *testing.T) {
		spec := PodSpec{Env: []corev1.EnvVar{{Name: "B", Value: "custom"}}}
		spec.AppendEnv(base)

		assert.Len(t, base, 1)
	})
}

func TestAppendVolumes(t *testing.T) {
	base := []corev1.Volume{{Name: "a"}}

	t.Run("empty", func(t *testing.T) {
		assert.Equal(t, base, PodSpec{}.AppendVolumes(base))
	})

	t.Run("appends after base", func(t *testing.T) {
		spec := PodSpec{Volumes: types.Volumes{{Name: "b", AllowedVolumeSource: types.AllowedVolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}}}}

		volumes := spec.AppendVolumes(base)
		assert.Len(t, volumes, 2)
		assert.Equal(t, "a", volumes[0].Name)
		assert.Equal(t, "b", volumes[1].Name)
		assert.NotNil(t, volumes[1].EmptyDir)
	})
}

func TestAppendVolumeMounts(t *testing.T) {
	base := []corev1.VolumeMount{{Name: "a", MountPath: "/a"}}

	t.Run("empty", func(t *testing.T) {
		assert.Equal(t, base, PodSpec{}.AppendVolumeMounts(base))
	})

	t.Run("appends after base", func(t *testing.T) {
		spec := PodSpec{VolumeMounts: []corev1.VolumeMount{{Name: "b", MountPath: "/b"}}}

		assert.Equal(t, []corev1.VolumeMount{{Name: "a", MountPath: "/a"}, {Name: "b", MountPath: "/b"}}, spec.AppendVolumeMounts(base))
	})
}

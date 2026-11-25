// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/types"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/prometheusmonitoring"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestConflictingVolumes(t *testing.T) {
	clt := fake.NewClient()
	emptyDir := types.AllowedVolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}

	components := map[string]struct {
		pod   func(pm *prometheusmonitoring.PrometheusMonitoring) *prometheusmonitoring.PodSpec
		names []string
		paths []string
	}{
		"targetAllocator": {func(pm *prometheusmonitoring.PrometheusMonitoring) *prometheusmonitoring.PodSpec {
			return &pm.Spec.TargetAllocator.PodSpec
		}, prometheusmonitoring.TargetAllocatorVolumeNames, prometheusmonitoring.TargetAllocatorVolumeMountPaths},
		"scraper": {func(pm *prometheusmonitoring.PrometheusMonitoring) *prometheusmonitoring.PodSpec {
			return &pm.Spec.Scraper.PodSpec
		}, prometheusmonitoring.ScraperVolumeNames, prometheusmonitoring.ScraperVolumeMountPaths},
		"gateway": {func(pm *prometheusmonitoring.PrometheusMonitoring) *prometheusmonitoring.PodSpec {
			return &pm.Spec.Gateway.PodSpec
		}, prometheusmonitoring.GatewayVolumeNames, prometheusmonitoring.GatewayVolumeMountPaths},
	}

	t.Run("no custom volumes", func(t *testing.T) {
		assert.Empty(t, conflictingVolumes(t.Context(), clt, newTestPM()))
	})

	t.Run("non conflicting custom volumes", func(t *testing.T) {
		for name, c := range components {
			pm := newTestPM()
			pod := c.pod(pm)
			pod.Volumes = types.Volumes{{Name: "custom", AllowedVolumeSource: emptyDir}}
			pod.VolumeMounts = []corev1.VolumeMount{{Name: "custom", MountPath: "/custom"}}

			assert.Empty(t, conflictingVolumes(t.Context(), clt, pm), name)
		}
	})

	t.Run("conflicting volume name", func(t *testing.T) {
		for name, c := range components {
			for _, reserved := range c.names {
				pm := newTestPM()
				c.pod(pm).Volumes = types.Volumes{{Name: reserved, AllowedVolumeSource: emptyDir}}

				msg := conflictingVolumes(t.Context(), clt, pm)

				assert.Contains(t, msg, name)
				assert.Contains(t, msg, reserved)
				assert.Contains(t, msg, "volume name")
			}
		}
	})

	t.Run("conflicting volume mount path", func(t *testing.T) {
		for name, c := range components {
			for _, reserved := range c.paths {
				pm := newTestPM()
				c.pod(pm).Volumes = types.Volumes{{Name: "custom", AllowedVolumeSource: emptyDir}}
				c.pod(pm).VolumeMounts = []corev1.VolumeMount{{Name: "custom", MountPath: reserved}}

				msg := conflictingVolumes(t.Context(), clt, pm)

				assert.Contains(t, msg, name)
				assert.Contains(t, msg, reserved)
				assert.Contains(t, msg, "volume mount path")
			}
		}
	})

	t.Run("volume mount nested in reserved path", func(t *testing.T) {
		for name, c := range components {
			pm := newTestPM()
			c.pod(pm).Volumes = types.Volumes{{Name: "custom", AllowedVolumeSource: emptyDir}}
			c.pod(pm).VolumeMounts = []corev1.VolumeMount{{Name: "custom", MountPath: c.paths[0] + "/nested"}}

			assert.Contains(t, conflictingVolumes(t.Context(), clt, pm), "volume mount path", name)
		}
	})

	t.Run("volume mount without matching volume", func(t *testing.T) {
		for name, c := range components {
			pm := newTestPM()
			c.pod(pm).VolumeMounts = []corev1.VolumeMount{{Name: "missing", MountPath: "/custom"}}

			msg := conflictingVolumes(t.Context(), clt, pm)

			assert.Contains(t, msg, name)
			assert.Contains(t, msg, "missing")
		}
	})
}

func newTestPM() *prometheusmonitoring.PrometheusMonitoring {
	return testPrometheusMonitoring.DeepCopy()
}

func TestValidateRejectsConflictingVolumes(t *testing.T) {
	validator := &Validator{apiReader: fake.NewClient()}
	pm := newTestPM()
	pm.Spec.Gateway.Volumes = types.Volumes{{Name: "custom", AllowedVolumeSource: types.AllowedVolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}}}
	pm.Spec.Gateway.VolumeMounts = []corev1.VolumeMount{{Name: "custom", MountPath: prometheusmonitoring.ConfigMountDir}}

	t.Run("ValidateCreate", func(t *testing.T) {
		_, err := validator.ValidateCreate(t.Context(), pm)

		require.Error(t, err)
		assert.Contains(t, err.Error(), prometheusmonitoring.ConfigMountDir)
	})

	t.Run("ValidateUpdate", func(t *testing.T) {
		_, err := validator.ValidateUpdate(t.Context(), testPrometheusMonitoring, pm)

		require.Error(t, err)
	})
}

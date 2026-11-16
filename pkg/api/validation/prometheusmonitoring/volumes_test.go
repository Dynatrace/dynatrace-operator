// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/types"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/prometheusmonitoring"
	pmconsts "github.com/Dynatrace/dynatrace-operator/pkg/controllers/prometheusmonitoring/consts"
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
		}, pmconsts.TargetAllocatorVolumeNames, pmconsts.TargetAllocatorVolumeMountPaths},
		"scraper": {func(pm *prometheusmonitoring.PrometheusMonitoring) *prometheusmonitoring.PodSpec {
			return &pm.Spec.Scraper.PodSpec
		}, pmconsts.ScraperVolumeNames, pmconsts.ScraperVolumeMountPaths},
		"gateway": {func(pm *prometheusmonitoring.PrometheusMonitoring) *prometheusmonitoring.PodSpec {
			return &pm.Spec.Gateway.PodSpec
		}, pmconsts.GatewayVolumeNames, pmconsts.GatewayVolumeMountPaths},
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
				c.pod(pm).VolumeMounts = []corev1.VolumeMount{{Name: "custom", MountPath: reserved}}

				msg := conflictingVolumes(t.Context(), clt, pm)

				assert.Contains(t, msg, name)
				assert.Contains(t, msg, reserved)
				assert.Contains(t, msg, "volume mount path")
			}
		}
	})
}

func newTestPM() *prometheusmonitoring.PrometheusMonitoring {
	return testPrometheusMonitoring.DeepCopy()
}

func TestValidateRejectsConflictingVolumes(t *testing.T) {
	validator := &Validator{apiReader: fake.NewClient()}
	pm := newTestPM()
	pm.Spec.Gateway.VolumeMounts = []corev1.VolumeMount{{Name: "custom", MountPath: pmconsts.ConfigMountDir}}

	t.Run("ValidateCreate", func(t *testing.T) {
		_, err := validator.ValidateCreate(t.Context(), pm)

		require.Error(t, err)
		assert.Contains(t, err.Error(), pmconsts.ConfigMountDir)
	})

	t.Run("ValidateUpdate", func(t *testing.T) {
		_, err := validator.ValidateUpdate(t.Context(), testPrometheusMonitoring, pm)

		require.Error(t, err)
	})
}

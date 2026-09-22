// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package statefulset

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/telemetryingest"
	"github.com/Dynatrace/dynatrace-operator/pkg/otelcgen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestProbes(t *testing.T) {
	expectedHTTPGet := &corev1.HTTPGetAction{
		Path: "/",
		Port: intstr.FromInt32(otelcgen.ExtensionsHealthCheckPort),
	}

	t.Run("liveness probe", func(t *testing.T) {
		probe := buildLivenessProbe()
		require.NotNil(t, probe)
		assert.Equal(t, expectedHTTPGet, probe.HTTPGet)
		assert.EqualValues(t, 10, probe.InitialDelaySeconds)
		assert.EqualValues(t, 30, probe.PeriodSeconds)
		assert.EqualValues(t, 3, probe.FailureThreshold)
		assert.EqualValues(t, 2, probe.TimeoutSeconds)
		assert.EqualValues(t, 1, probe.SuccessThreshold)
	})

	t.Run("readiness probe", func(t *testing.T) {
		probe := buildReadinessProbe()
		require.NotNil(t, probe)
		assert.Equal(t, expectedHTTPGet, probe.HTTPGet)
		assert.EqualValues(t, 5, probe.InitialDelaySeconds)
		assert.EqualValues(t, 10, probe.PeriodSeconds)
		assert.EqualValues(t, 3, probe.FailureThreshold)
		assert.EqualValues(t, 2, probe.TimeoutSeconds)
		assert.EqualValues(t, 1, probe.SuccessThreshold)
	})

	t.Run("probes are always set", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.TelemetryIngest = &telemetryingest.Spec{}

		container := getContainer(dk, 1)
		assert.NotNil(t, container.LivenessProbe)
		assert.NotNil(t, container.ReadinessProbe)
	})
}

func TestContainer(t *testing.T) {
	t.Run("builds the telemetry config arg with default (empty) extraArgs", func(t *testing.T) {
		assert.Equal(t, []string{"--config=file:///config/telemetry.yaml"}, buildArgs([]string{}))
	})
	t.Run("builds the telemetry config arg with (user-specified) extraArgs", func(t *testing.T) {
		assert.Equal(t, []string{"--config=file:///config/telemetry.yaml", "--set=exporters::debug::verbosity=basic"}, buildArgs([]string{"--set=exporters::debug::verbosity=basic"}))
	})
	t.Run("nil extraArgs produces only base arg", func(t *testing.T) {
		assert.Equal(t, []string{"--config=file:///config/telemetry.yaml"}, buildArgs(nil))
	})
	t.Run("multiple extraArgs are appended in order", func(t *testing.T) {
		assert.Equal(t, []string{
			"--config=file:///config/telemetry.yaml",
			"--set=exporters::debug::verbosity=basic",
			"--feature-gates=+component.UseLocalHostAsDefaultHost",
		}, buildArgs([]string{
			"--set=exporters::debug::verbosity=basic",
			"--feature-gates=+component.UseLocalHostAsDefaultHost",
		}))
	})
	t.Run("args are wired into container", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.TelemetryIngest = &telemetryingest.Spec{}
		dk.Spec.Templates.OpenTelemetryCollector.Args = []string{"--set=exporters::debug::verbosity=basic"}

		container := getContainer(dk, 1)
		assert.Equal(t, []string{
			"--config=file:///config/telemetry.yaml",
			"--set=exporters::debug::verbosity=basic",
		}, container.Args)
	})
}

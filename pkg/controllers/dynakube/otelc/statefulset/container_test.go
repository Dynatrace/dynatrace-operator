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
	})

	t.Run("readiness probe", func(t *testing.T) {
		probe := buildReadinessProbe()
		require.NotNil(t, probe)
		assert.Equal(t, expectedHTTPGet, probe.HTTPGet)
		assert.EqualValues(t, 5, probe.InitialDelaySeconds)
		assert.EqualValues(t, 10, probe.PeriodSeconds)
	})
}

func TestContainer(t *testing.T) {
	t.Run("only TelemetryIngest enabled", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.TelemetryIngest = &telemetryingest.Spec{}

		assert.Equal(t, []string{"--config=file:///config/telemetry.yaml"}, buildArgs(dk))
	})

	t.Run("only EEC enabled", func(t *testing.T) {
		dk := getTestDynakubeWithExtensions()
		assert.Equal(
			t,
			[]string{"--config=eec://dynakube-extension-controller.dynatrace:14599/otcconfig/prometheusMetrics#refresh-interval=5s&auth-file=/secrets/tokens/datasource.token"},
			buildArgs(dk),
		)
	})

	t.Run("TelemetryIngest and EEC enabled", func(t *testing.T) {
		dk := getTestDynakubeWithExtensions()
		dk.Spec.TelemetryIngest = &telemetryingest.Spec{}

		assert.Equal(
			t,
			[]string{
				"--config=eec://dynakube-extension-controller.dynatrace:14599/otcconfig/prometheusMetrics#refresh-interval=5s&auth-file=/secrets/tokens/datasource.token",
				"--config=file:///config/telemetry.yaml",
			},
			buildArgs(dk),
		)
	})
}

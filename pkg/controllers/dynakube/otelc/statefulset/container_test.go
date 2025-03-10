package statefulset

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube/telemetryingest"
	"github.com/stretchr/testify/assert"
)

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
			[]string{"--config=eec://dynakube-extensions-controller.dynatrace:14599/otcconfig/prometheusMetrics#refresh-interval=5s&auth-file=/secrets/tokens/otelc.token"},
			buildArgs(dk),
		)
	})

	t.Run("TelemetryIngest and EEC enabled", func(t *testing.T) {
		dk := getTestDynakubeWithExtensions()
		dk.Spec.TelemetryIngest = &telemetryingest.Spec{}

		assert.Equal(
			t,
			[]string{
				"--config=eec://dynakube-extensions-controller.dynatrace:14599/otcconfig/prometheusMetrics#refresh-interval=5s&auth-file=/secrets/tokens/otelc.token",
				"--config=file:///config/telemetry.yaml",
			},
			buildArgs(dk),
		)
	})
}

package statefulset

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/telemetryservice"
	"github.com/stretchr/testify/assert"
)

func TestContainer(t *testing.T) {
	t.Run("only TelemetryService enabled", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.TelemetryService = &telemetryservice.Spec{}

		assert.Equal(t, []string{"--config=file:///osconfig/config.yaml"}, buildArgs(dk))
	})

	t.Run("only EEC enabled", func(t *testing.T) {
		dk := getTestDynakubeWithExtensions()
		assert.Equal(
			t,
			[]string{"--config=eec://dynakube-extensions-controller.dynatrace:14599/otcconfig/prometheusMetrics#refresh-interval=5s&auth-file=/secrets/tokens/otelc.token"},
			buildArgs(dk),
		)
	})

	t.Run("TelemetryService and EEC enabled", func(t *testing.T) {
		dk := getTestDynakubeWithExtensions()
		dk.Spec.TelemetryService = &telemetryservice.Spec{}

		assert.Equal(
			t,
			[]string{
				"--config=eec://dynakube-extensions-controller.dynatrace:14599/otcconfig/prometheusMetrics#refresh-interval=5s&auth-file=/secrets/tokens/otelc.token",
				"--config=file:///osconfig/config.yaml",
			},
			buildArgs(dk),
		)
	})
}

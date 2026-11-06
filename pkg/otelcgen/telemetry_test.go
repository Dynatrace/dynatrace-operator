// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package otelcgen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/confmap"
)

func TestNewMetricsTelemetry(t *testing.T) {
	conf := confmap.New()
	require.NoError(t, conf.Marshal(&Config{Service: ServiceConfig{Telemetry: NewMetricsTelemetry("http://dk-self-monitoring:4318")}}))

	telemetry, ok := conf.ToStringMap()["service"].(map[string]any)["telemetry"].(map[string]any)
	require.True(t, ok)

	t.Run("exports metrics via otlp http", func(t *testing.T) {
		metrics, ok := telemetry["metrics"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "Normal", metrics["level"])

		readers, ok := metrics["readers"].([]any)
		require.True(t, ok)
		require.Len(t, readers, 1)

		otlp, ok := readers[0].(map[string]any)["periodic"].(map[string]any)["exporter"].(map[string]any)["otlp"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "http://dk-self-monitoring:4318", otlp["endpoint"])
		assert.Equal(t, "http/protobuf", otlp["protocol"])
		assert.Equal(t, "delta", otlp["temporality_preference"])
	})

	t.Run("keeps default logs settings", func(t *testing.T) {
		logs, ok := telemetry["logs"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "console", logs["encoding"])
		assert.Equal(t, []any{"stderr"}, logs["output_paths"])
		assert.Equal(t, []any{"stderr"}, logs["error_output_paths"])
	})
}

// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package otelcgen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pipeline"
	"gopkg.in/yaml.v3"
)

func TestNewTelemetryIngest(t *testing.T) {
	cfg := NewTelemetryIngest(TelemetryIngestSpec{
		Protocols:        RegisteredProtocols,
		Endpoint:         "exporters-test-endpoint",
		APIToken:         "test-token",
		CAFile:           "/run/opensignals/cacerts/certs",
		IncludeSystemCAs: true,
		TLSCert:          "/run/opensignals/tls/tls.crt",
		TLSKey:           "/run/opensignals/tls/tls.key",
	})

	c, err := cfg.Marshal()
	require.NoError(t, err)

	expectedOutput, err := os.ReadFile(filepath.Join("testdata", "full_config.yaml"))
	require.NoError(t, err)

	assert.YAMLEq(t, string(expectedOutput), string(c))
}

func TestNewTelemetryIngestServices(t *testing.T) {
	tests := []struct {
		name      string
		protocols []Protocol
		golden    string
	}{
		{name: "all protocols", protocols: RegisteredProtocols, golden: "services_only.yaml"},
		{name: "zipkin only", protocols: []Protocol{ZipkinProtocol}, golden: "services_zipkin_only.yaml"},
		{name: "statsd only", protocols: []Protocol{StatsdProtocol}, golden: "services_statsd_only.yaml"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := NewTelemetryIngest(TelemetryIngestSpec{Protocols: tt.protocols}).Marshal()
			require.NoError(t, err)

			expectedOutput, err := os.ReadFile(filepath.Join("testdata", tt.golden))
			require.NoError(t, err)

			assert.YAMLEq(t, serviceSection(t, expectedOutput), serviceSection(t, c))
		})
	}
}

var logsPipeline = pipeline.NewID(pipeline.SignalLogs)

func TestNewTelemetryIngestPipelineProcessors(t *testing.T) {
	t.Run("with resource attributes", func(t *testing.T) {
		cfg := NewTelemetryIngest(TelemetryIngestSpec{
			Protocols:          []Protocol{OTLPProtocol},
			ResourceAttributes: map[string]string{"team": "shopping-cart"},
		})

		require.Equal(t, []component.ID{
			MemoryLimiterID, transformPodIP, K8sAttributesAnnotationsID, StaticResourceAttrsID, K8sAttributesFactsID, TransformID, batchLogs,
		}, cfg.Service.Pipelines[logsPipeline].Processors)
	})

	t.Run("without resource attributes", func(t *testing.T) {
		cfg := NewTelemetryIngest(TelemetryIngestSpec{Protocols: []Protocol{OTLPProtocol}})

		require.Equal(t, []component.ID{
			MemoryLimiterID, transformPodIP, K8sAttributesAnnotationsID, K8sAttributesFactsID, TransformID, batchLogs,
		}, cfg.Service.Pipelines[logsPipeline].Processors)
	})
}

// serviceSection returns the service section of the marshaled config, as the goldens of the service only
// have empty receivers, processors and exporters.
func serviceSection(t *testing.T, cfg []byte) string {
	t.Helper()

	var parsed map[string]any

	require.NoError(t, yaml.Unmarshal(cfg, &parsed))

	out, err := yaml.Marshal(parsed["service"])
	require.NoError(t, err)

	return string(out)
}

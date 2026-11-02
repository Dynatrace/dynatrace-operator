// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package selfmonitoring

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"sigs.k8s.io/yaml"
)

func TestBuildConfigData(t *testing.T) {
	t.Run("exports to the tenant", func(t *testing.T) {
		dk := &dynakube.DynaKube{Namespace: "dynatrace", Spec: dynakube.DynaKubeSpec{APIURL: "https://tenant.live.dynatrace.com/api"}}

		data := buildConfigData(dk)

		assert.Equal(t, "https://tenant.live.dynatrace.com/api/v2/otlp", data.Endpoint)
		assert.Equal(t, "dynatrace", data.Namespace)
		assert.Empty(t, data.CustomCAPath)
		assert.Empty(t, data.ResourceAttributes)
	})

	t.Run("resource attributes", func(t *testing.T) {
		attrs := map[string]string{"cost-center": "42"}
		dk := &dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{ResourceAttributes: attrs}}

		assert.Equal(t, attrs, buildConfigData(dk).ResourceAttributes)
	})

	t.Run("trusted CAs", func(t *testing.T) {
		dk := &dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{TrustedCAs: "my-cas"}}

		assert.Equal(t, "/tls/custom/cacerts/rootca.pem", buildConfigData(dk).CustomCAPath)
	})
}

func TestBuildSelfMonitoringOTelConfig(t *testing.T) {
	data := selfMonitoringConfigData{Endpoint: "https://tenant/api/v2/otlp", Namespace: "dynatrace"}
	cfg := buildSelfMonitoringOTelConfig(data)

	t.Run("every pipeline component is defined", func(t *testing.T) {
		pipeline := cfg.Service.Pipelines[metricsSignalID]
		require.NotNil(t, pipeline)

		assertDefined := func(kind string, ids []component.ID, defined map[component.ID]component.Config) {
			for _, id := range ids {
				assert.Contains(t, defined, id, "%s %s is used in the pipeline but not defined", kind, id)
			}
		}
		assertDefined("receiver", pipeline.Receivers, cfg.Receivers)
		assertDefined("processor", pipeline.Processors, cfg.Processors)
		assertDefined("exporter", pipeline.Exporters, cfg.Exporters)

		for _, id := range cfg.Service.Extensions {
			assert.Contains(t, cfg.Extensions, id)
		}
	})

	t.Run("exporter authenticates with the mounted data ingest token", func(t *testing.T) {
		exporter := cfg.Exporters[otlpHTTPID].(map[string]any)
		assert.Equal(t, data.Endpoint, exporter["endpoint"])
		assert.Equal(t, bearerAuthID.String(), exporter["auth"].(map[string]any)["authenticator"])
		assert.NotContains(t, exporter, "tls")

		auth := cfg.Extensions[bearerAuthID].(map[string]any)
		assert.Equal(t, "/var/lib/dynatrace/secrets/tokens/data-ingest-token", auth["filename"])
		assert.Equal(t, "Api-Token", auth["scheme"])
	})

	t.Run("custom CA is passed to the exporter", func(t *testing.T) {
		exporter := buildOTLPHTTPExporter(selfMonitoringConfigData{CustomCAPath: "/ca.pem"}).(map[string]any)
		assert.Equal(t, map[string]any{"ca_file": "/ca.pem"}, exporter["tls"])
	})

	t.Run("k8s attributes are restricted to the DynaKube namespace", func(t *testing.T) {
		processor := cfg.Processors[k8sAttributesID].(map[string]any)
		assert.Equal(t, map[string]any{"namespace": "dynatrace"}, processor["filter"])
	})

	t.Run("receiver only serves OTLP/HTTP", func(t *testing.T) {
		protocols := cfg.Receivers[otlpID].(map[string]any)["protocols"].(map[string]any)
		assert.Equal(t, map[string]any{"http": map[string]any{"endpoint": "${env:MY_POD_IP}:4318"}}, protocols)
	})

	t.Run("does not convert to delta", func(t *testing.T) {
		rendered, err := renderConfig(data)
		require.NoError(t, err)
		assert.NotContains(t, rendered, "cumulative_to_delta")
	})

	t.Run("no resource attributes processor without resource attributes", func(t *testing.T) {
		assert.NotContains(t, cfg.Processors, resourceAttrsID)
		assert.NotContains(t, cfg.Service.Pipelines[metricsSignalID].Processors, resourceAttrsID)
	})

	t.Run("resource attributes are inserted last without overwriting", func(t *testing.T) {
		cfg := buildSelfMonitoringOTelConfig(selfMonitoringConfigData{ResourceAttributes: map[string]string{"b": "2", "a": "1"}})

		processors := cfg.Service.Pipelines[metricsSignalID].Processors
		assert.Equal(t, resourceAttrsID, processors[len(processors)-1])
		assert.Equal(t, map[string]any{"attributes": []map[string]any{
			{"key": "a", "value": "1", "action": "insert"},
			{"key": "b", "value": "2", "action": "insert"},
		}}, cfg.Processors[resourceAttrsID])
	})

	t.Run("marshals to valid yaml", func(t *testing.T) {
		rendered, err := renderConfig(data)
		require.NoError(t, err)

		var parsed map[string]any
		require.NoError(t, yaml.Unmarshal([]byte(rendered), &parsed))
		assert.Contains(t, parsed, "service")
		assert.Contains(t, rendered, "transform/selfmon_rename")
	})
}

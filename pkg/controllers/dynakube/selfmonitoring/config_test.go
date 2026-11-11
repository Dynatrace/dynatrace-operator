// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package selfmonitoring

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

const testEndpoint = "https://tenant.live.dynatrace.com/api/v2/otlp"

// goldenConfig reads a golden ConfigMap fixture and returns its collector config data key,
// so config-rendering tests can compare directly against the same fixtures used
// by the reconciler tests instead of duplicating expected output.
func goldenConfig(t *testing.T, name string) string {
	t.Helper()

	b, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)

	cm := &corev1.ConfigMap{}
	require.NoError(t, yaml.Unmarshal(b, cm))

	return cm.Data[configKey]
}

func TestBuildSelfMonitoringOTelConfig(t *testing.T) {
	t.Run("every pipeline component is defined", func(t *testing.T) {
		cfg := buildSelfMonitoringOTelConfig(selfMonitoringConfigData{ResourceAttributes: map[string]string{"a": "1"}})

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
}

func TestRenderConfig(t *testing.T) {
	t.Run("default config matches golden configmap", func(t *testing.T) {
		rendered, err := renderConfig(selfMonitoringConfigData{Endpoint: testEndpoint, Namespace: testNamespace})
		require.NoError(t, err)
		assert.YAMLEq(t, goldenConfig(t, "configmap.yaml"), rendered)
	})

	t.Run("custom CA and sorted resource attributes match golden configmap", func(t *testing.T) {
		data := selfMonitoringConfigData{
			Endpoint:           testEndpoint,
			Namespace:          testNamespace,
			CustomCAPath:       "/tls/custom/cacerts/rootca.pem",
			ResourceAttributes: map[string]string{"team": "platform", "cost-center": "42"},
		}
		rendered, err := renderConfig(data)
		require.NoError(t, err)
		assert.YAMLEq(t, goldenConfig(t, "configmap_with_trusted_cas_and_resource_attributes.yaml"), rendered)
	})
}

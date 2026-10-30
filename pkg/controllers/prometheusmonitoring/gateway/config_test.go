// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package gateway

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/otelcgen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pipeline"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

// goldenRelay reads a golden ConfigMap fixture and returns its "relay" data key,
// so config-rendering tests can compare directly against the same fixtures used
// by the reconciler tests instead of duplicating expected output.
func goldenRelay(t *testing.T, name string) string {
	t.Helper()

	b, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)

	cm := &corev1.ConfigMap{}
	require.NoError(t, yaml.Unmarshal(b, cm))

	return cm.Data[gatewayConfigKey]
}

func TestBuildGatewayOTelConfig(t *testing.T) {
	const endpoint = "https://abc12345.live.dynatrace.com/api/v2/otlp"

	processorsOf := func(t *testing.T, data gatewayConfigData) []component.ID {
		t.Helper()

		cfg := buildConfig(data)

		p := cfg.Service.Pipelines[pipeline.NewID(pipeline.SignalMetrics)]
		require.NotNil(t, p)

		return p.Processors
	}

	t.Run("pipeline without resource attributes", func(t *testing.T) {
		assert.Equal(t, []component.ID{
			otelcgen.MemoryLimiterID,
			metricStartTimeID,
			otelcgen.CumulativeToDeltaID,
			otelcgen.K8sAttributesAnnotationsID,
			otelcgen.K8sAttributesFactsID,
			otelcgen.TransformID,
		}, processorsOf(t, gatewayConfigData{Endpoint: endpoint}))
	})

	t.Run("static resource attributes are between the annotations and the facts", func(t *testing.T) {
		data := gatewayConfigData{Endpoint: endpoint, ResourceAttributes: map[string]string{"team": "a"}}

		assert.Equal(t, []component.ID{
			otelcgen.MemoryLimiterID,
			metricStartTimeID,
			otelcgen.CumulativeToDeltaID,
			otelcgen.K8sAttributesAnnotationsID,
			otelcgen.StaticResourceAttrsID,
			otelcgen.K8sAttributesFactsID,
			otelcgen.TransformID,
		}, processorsOf(t, data))
	})

	t.Run("facts include the workload UIDs and the container name", func(t *testing.T) {
		cfg := buildConfig(gatewayConfigData{Endpoint: endpoint})

		facts, ok := cfg.Processors[otelcgen.K8sAttributesFactsID].(map[string]any)
		require.True(t, ok)

		extract, ok := facts["extract"].(map[string]any)
		require.True(t, ok)
		assert.Subset(t, extract["metadata"], []string{"k8s.deployment.uid", "k8s.job.uid", "k8s.container.name"})
		assert.NotContains(t, extract, "annotations")
	})
}

func TestBuildTransformConfig(t *testing.T) {
	cfg, ok := buildTransformConfig().(map[string]any)
	require.True(t, ok)

	assert.Equal(t, "ignore", cfg["error_mode"])
	assert.NotContains(t, cfg, "trace_statements")
	assert.NotContains(t, cfg, "log_statements")

	groups, ok := cfg["metric_statements"].([]map[string]any)
	require.True(t, ok)

	var all []string

	for _, group := range groups {
		statements, ok := group["statements"].([]string)
		require.True(t, ok)

		all = append(all, statements...)
	}

	t.Run("the annotations are merged first", func(t *testing.T) {
		assert.Equal(t, otelcgen.MergeDynatraceAnnotations(), all[:2])
	})

	t.Run("prometheus labels are renamed on the data points", func(t *testing.T) {
		assert.Equal(t, "datapoint", groups[1]["context"])
		assert.Contains(t, all, `set(attributes["http.request.method"], attributes["http_request_method"]) where attributes["http_request_method"] != nil`)
		assert.Contains(t, all, `delete_key(attributes, "http_request_method")`)
	})

	t.Run("workload attributes use the guarded statements incl. the uid", func(t *testing.T) {
		for _, statement := range otelcgen.WorkloadFromOwners(true) {
			assert.Contains(t, all, statement)
		}
	})

	t.Run("internal keys are deleted", func(t *testing.T) {
		assert.Contains(t, all, `delete_key(attributes, "otel.scope.name")`)
		assert.Contains(t, all, `delete_key(attributes, "processor")`)
	})

	t.Run("the cluster name falls back to the env var, but no other cluster or dt attributes are set", func(t *testing.T) {
		assert.Contains(t, all, `set(attributes["k8s.cluster.name"], "${env:K8S_CLUSTER_NAME}") where attributes["k8s.cluster.name"] == nil and Len("${env:K8S_CLUSTER_NAME}") > 0`)

		for _, statement := range all {
			assert.NotContains(t, statement, "dt.")
			assert.NotContains(t, statement, "k8s.cluster.uid")
		}
	})
}

func TestRenderGatewayConfig_ResourceAttributes(t *testing.T) {
	const endpoint = "https://abc12345.live.dynatrace.com/api/v2/otlp"

	t.Run("empty resource attributes matches golden configmap without attributes", func(t *testing.T) {
		rendered, err := renderGatewayConfig(gatewayConfigData{Endpoint: endpoint})
		require.NoError(t, err)
		assert.YAMLEq(t, goldenRelay(t, "configmap.yaml"), rendered)
	})

	t.Run("resource attributes rendered as sorted baseline set statements matches golden configmap with attributes", func(t *testing.T) {
		data := gatewayConfigData{
			Endpoint: endpoint,
			ResourceAttributes: map[string]string{
				"favorite.coffee": "espresso",
				"deploy.mood":     "yolo",
			},
		}
		rendered, err := renderGatewayConfig(data)
		require.NoError(t, err)
		assert.YAMLEq(t, goldenRelay(t, "configmap_with_resource_attributes.yaml"), rendered)
	})
}

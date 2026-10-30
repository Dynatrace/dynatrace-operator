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
)

// telemetryIngestProcessors returns the processors that are registered for telemetryIngest.
func telemetryIngestProcessors(spec TelemetryIngestSpec) map[component.ID]component.Config {
	return NewBuilder().telemetryIngestProcessors(spec).Build().Processors
}

func TestBuilderTelemetryIngestProcessors(t *testing.T) {
	cfg := NewBuilder().telemetryIngestProcessors(TelemetryIngestSpec{}).Build()
	c, err := cfg.Marshal()
	require.NoError(t, err)

	expectedOutput, err := os.ReadFile(filepath.Join("testdata", "processors_only.yaml"))
	require.NoError(t, err)

	assert.YAMLEq(t, string(expectedOutput), string(c))
}

func TestBuildProcessors_WithResourceAttributes(t *testing.T) {
	t.Run("no static resource attrs processor when empty", func(t *testing.T) {
		processors := telemetryIngestProcessors(TelemetryIngestSpec{})

		assert.NotContains(t, processors, StaticResourceAttrsID)
	})

	t.Run("static resource attrs processor added, sorted, insert action", func(t *testing.T) {
		processors := telemetryIngestProcessors(TelemetryIngestSpec{
			ResourceAttributes: map[string]string{"team": "shopping-cart", "env": "prod"},
		})

		staticAttrs, ok := processors[StaticResourceAttrsID].(map[string]any)
		require.True(t, ok)

		attributes, ok := staticAttrs["attributes"].([]map[string]any)
		require.True(t, ok)
		require.Len(t, attributes, 2)

		assert.Equal(t, map[string]any{"key": "env", "value": "prod", "action": "insert"}, attributes[0])
		assert.Equal(t, map[string]any{"key": "team", "value": "shopping-cart", "action": "insert"}, attributes[1])
	})
}

func TestK8sAttributesJSONAnnotation(t *testing.T) {
	processors := telemetryIngestProcessors(TelemetryIngestSpec{})

	k8sAttr, ok := processors[K8sAttributesAnnotationsID].(map[string]any)
	require.True(t, ok)

	extract, ok := k8sAttr["extract"].(map[string]any)
	require.True(t, ok)

	// k8sattributesprocessor treats an omitted "metadata" field as "use its built-in default
	// list" rather than "extract nothing", so it must be explicitly emptied here - otherwise
	// this instance would also claim built-in facts (e.g. k8s.namespace.name) before
	// resource/staticAttrs gets a chance to run.
	assert.Equal(t, []string{}, extract["metadata"])

	annotations, ok := extract["annotations"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, annotations, 2)

	t.Run("regex rule kept", func(t *testing.T) {
		rule := annotations[0]
		assert.Equal(t, "pod", rule["from"])
		assert.Equal(t, "metadata.dynatrace.com/(.*)", rule["key_regex"])
		assert.Equal(t, "$$1", rule["tag_name"])
		assert.NotContains(t, rule, "key")
	})

	t.Run("exact-key rule added", func(t *testing.T) {
		rule := annotations[1]
		assert.Equal(t, "pod", rule["from"])
		assert.Equal(t, "metadata.dynatrace.com", rule["key"])
		assert.Equal(t, "metadata.dynatrace.com", rule["tag_name"])
		assert.NotContains(t, rule, "key_regex")
	})
}

func TestK8sAttributesFacts(t *testing.T) {
	processors := telemetryIngestProcessors(TelemetryIngestSpec{})

	k8sAttr, ok := processors[K8sAttributesFactsID].(map[string]any)
	require.True(t, ok)

	extract, ok := k8sAttr["extract"].(map[string]any)
	require.True(t, ok)

	assert.NotContains(t, extract, "annotations")
	assert.Equal(t, K8sFactsMetadata(false), extract["metadata"])
}

func TestDynatraceTransformationsJSONAnnotation(t *testing.T) {
	transformations := dynatraceTransformations()
	require.Len(t, transformations, 1)

	statements, ok := transformations[0]["statements"].([]string)
	require.True(t, ok)
	require.GreaterOrEqual(t, len(statements), 2)

	t.Run("merge_maps statement is first", func(t *testing.T) {
		assert.Contains(t, statements[0], "merge_maps(attributes, ParseJSON(attributes[\"metadata.dynatrace.com\"]), \"upsert\")")
		assert.Contains(t, statements[0], `IsMatch(attributes["metadata.dynatrace.com"], "^\\{")`)
	})

	t.Run("delete_key statement is second", func(t *testing.T) {
		assert.Equal(t, `delete_key(attributes, "metadata.dynatrace.com")`, statements[1])
	})

	t.Run("all signal types carry JSON annotation statements", func(t *testing.T) {
		transform := buildTransform()

		for _, signalKey := range []string{"log_statements", "metric_statements", "trace_statements"} {
			stmts, ok := transform[signalKey].([]map[string]any)
			require.True(t, ok, "expected []map[string]any for %s", signalKey)
			require.Len(t, stmts, 1)

			block, ok := stmts[0]["statements"].([]string)
			require.True(t, ok)
			assert.Contains(t, block[0], "merge_maps", "%s missing merge_maps", signalKey)
			assert.Equal(t, `delete_key(attributes, "metadata.dynatrace.com")`, block[1], "%s missing delete_key", signalKey)
		}
	})
}

func TestMemoryLimiter(t *testing.T) {
	assert.Equal(
		t,
		map[string]any{"check_interval": "1s", "limit_percentage": uint32(95), "spike_limit_percentage": uint32(5)},
		MemoryLimiter(MemoryLimiterOpts{LimitPercentage: 95, SpikePercentage: 5}),
	)
}

func TestPodAssociation(t *testing.T) {
	got := PodAssociation()

	require.Len(t, got, 4)
	assert.Equal(t, []map[string]any{{"from": "connection"}}, got[3]["sources"])
}

func TestK8sAttributes(t *testing.T) {
	t.Run("nil metadata is omitted so the processor uses its defaults", func(t *testing.T) {
		cfg, ok := K8sAttributes(K8sAttributesOpts{}).(map[string]any)
		require.True(t, ok)

		assert.Equal(t, map[string]any{}, cfg["extract"])
		assert.Equal(t, PodAssociation(), cfg["pod_association"])
	})

	t.Run("empty metadata is kept so the processor extracts nothing", func(t *testing.T) {
		cfg, ok := K8sAttributes(K8sAttributesOpts{Metadata: []string{}}).(map[string]any)
		require.True(t, ok)

		assert.Equal(t, map[string]any{"metadata": []string{}}, cfg["extract"])
	})

	t.Run("annotations are only extracted when requested", func(t *testing.T) {
		with, ok := K8sAttributes(K8sAttributesOpts{Annotations: true}).(map[string]any)
		require.True(t, ok)
		assert.Contains(t, with["extract"], "annotations")

		without, ok := K8sAttributes(K8sAttributesOpts{}).(map[string]any)
		require.True(t, ok)
		assert.NotContains(t, without["extract"], "annotations")
	})
}

func TestK8sFactsMetadata(t *testing.T) {
	without := K8sFactsMetadata(false)
	with := K8sFactsMetadata(true)

	assert.NotContains(t, without, "k8s.deployment.uid")
	assert.Equal(t, without, with[:len(without)], "UIDs are only appended")
	assert.ElementsMatch(t, []string{
		"k8s.deployment.uid", "k8s.replicaset.uid", "k8s.statefulset.uid",
		"k8s.daemonset.uid", "k8s.cronjob.uid", "k8s.job.uid",
	}, with[len(without):])

	// callers may modify the result
	without[0] = "changed"
	assert.NotEqual(t, "changed", K8sFactsMetadata(false)[0])
}

func TestDynatraceTransformationsClusterAttributes(t *testing.T) {
	getStatements := func() []string {
		transformations := dynatraceTransformations()
		require.Len(t, transformations, 1)

		statements, ok := transformations[0]["statements"].([]string)
		require.True(t, ok)

		return statements
	}

	t.Run("enabled", func(t *testing.T) {
		statements := getStatements()

		assert.Contains(t, statements, setLiteralIfAbsent("k8s.cluster.uid", "${env:K8S_CLUSTER_UID}"))
		assert.Contains(t, statements, setLiteralIfAbsent("k8s.cluster.name", "${env:K8S_CLUSTER_NAME}"))
		assert.Contains(t, statements, `set(attributes["dt.kubernetes.workload.name"], attributes["k8s.workload.name"])`)
		assert.Contains(t, statements, `set(attributes["dt.entity.kubernetes_cluster"], "${env:DT_ENTITY_KUBERNETES_CLUSTER}")`)
	})

	t.Run("owner facts are deleted after the cluster attributes are set", func(t *testing.T) {
		statements := getStatements()

		assert.Equal(t, DeleteOwnerFacts(false), statements[len(statements)-len(DeleteOwnerFacts(false)):])
	})
}

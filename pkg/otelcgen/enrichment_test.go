// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package otelcgen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pipeline"
)

func transformGroups(t *testing.T, cfg *Config, key string) []map[string]any {
	t.Helper()

	transform, ok := cfg.Processors[TransformID].(map[string]any)
	require.True(t, ok)

	groups, ok := transform[key].([]map[string]any)
	require.True(t, ok, "expected []map[string]any for %s", key)

	return groups
}

func statementsOf(t *testing.T, group map[string]any) []string {
	t.Helper()

	statements, ok := group["statements"].([]string)
	require.True(t, ok)

	return statements
}

func TestK8sEnrichment(t *testing.T) {
	t.Run("processors are registered in the order of the precedence", func(t *testing.T) {
		b := NewBuilder().
			Receiver(OTLPID, Empty).
			Exporter(OTLPHTTPID, Empty).
			K8sEnrichment(K8sEnrichmentOpts{ResourceAttributes: map[string]string{"team": "a"}})

		assert.Equal(t,
			[]component.ID{K8sAttributesAnnotationsID, StaticResourceAttrsID, K8sAttributesFactsID, TransformID},
			b.Build(pipeline.SignalMetrics).Service.Pipelines[pipeline.NewID(pipeline.SignalMetrics)].Processors,
		)
	})

	t.Run("no static resource attributes processor without attributes", func(t *testing.T) {
		cfg := NewBuilder().K8sEnrichment(K8sEnrichmentOpts{}).Build()

		assert.NotContains(t, cfg.Processors, StaticResourceAttrsID)
	})

	t.Run("static resource attributes are sorted and inserted", func(t *testing.T) {
		cfg := NewBuilder().K8sEnrichment(K8sEnrichmentOpts{
			ResourceAttributes: map[string]string{"team": "shopping-cart", "env": "prod"},
		}).Build()

		staticAttrs, ok := cfg.Processors[StaticResourceAttrsID].(map[string]any)
		require.True(t, ok)

		assert.Equal(t, []map[string]any{
			{"key": "env", "value": "prod", "action": "insert"},
			{"key": "team", "value": "shopping-cart", "action": "insert"},
		}, staticAttrs["attributes"])
	})

	t.Run("annotations instance extracts no metadata, facts instance no annotations", func(t *testing.T) {
		cfg := NewBuilder().K8sEnrichment(K8sEnrichmentOpts{}).Build()

		annotations, ok := cfg.Processors[K8sAttributesAnnotationsID].(map[string]any)
		require.True(t, ok)

		extract, ok := annotations["extract"].(map[string]any)
		require.True(t, ok)
		// an omitted metadata makes the processor use its built-in defaults, so it must be explicitly empty
		assert.Equal(t, []string{}, extract["metadata"])
		assert.Contains(t, extract, "annotations")

		facts, ok := cfg.Processors[K8sAttributesFactsID].(map[string]any)
		require.True(t, ok)

		extract, ok = facts["extract"].(map[string]any)
		require.True(t, ok)
		assert.NotContains(t, extract, "annotations")
		assert.Equal(t, k8sFactsMetadata(false), extract["metadata"])
	})

	t.Run("workload UIDs and extra facts", func(t *testing.T) {
		cfg := NewBuilder().K8sEnrichment(K8sEnrichmentOpts{
			WorkloadUIDs:       true,
			ExtraFactsMetadata: []string{"k8s.container.name"},
		}).Build()

		facts, ok := cfg.Processors[K8sAttributesFactsID].(map[string]any)
		require.True(t, ok)

		extract, ok := facts["extract"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, append(k8sFactsMetadata(true), "k8s.container.name"), extract["metadata"])

		statements := statementsOf(t, transformGroups(t, cfg, "metric_statements")[0])
		for _, statement := range workloadFromOwners(true) {
			assert.Contains(t, statements, statement)
		}
	})
}

func TestK8sEnrichmentTransform(t *testing.T) {
	t.Run("all signals by default", func(t *testing.T) {
		cfg := NewBuilder().K8sEnrichment(K8sEnrichmentOpts{}).Build()

		for _, key := range []string{"log_statements", "metric_statements", "trace_statements"} {
			groups := transformGroups(t, cfg, key)
			require.Len(t, groups, 1)

			statements := statementsOf(t, groups[0])
			assert.Equal(t, mergeDynatraceAnnotations(), statements[:2], key)
		}
	})

	t.Run("only the requested signals", func(t *testing.T) {
		cfg := NewBuilder().K8sEnrichment(K8sEnrichmentOpts{Signals: []pipeline.Signal{pipeline.SignalMetrics}}).Build()

		transform, ok := cfg.Processors[TransformID].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "ignore", transform["error_mode"])
		assert.Contains(t, transform, "metric_statements")
		assert.NotContains(t, transform, "trace_statements")
		assert.NotContains(t, transform, "log_statements")
	})

	t.Run("the merge comes first and the owner facts are deleted last in the resource group", func(t *testing.T) {
		cfg := NewBuilder().K8sEnrichment(K8sEnrichmentOpts{ResourceStatements: []string{"extra"}}).Build()

		statements := statementsOf(t, transformGroups(t, cfg, "metric_statements")[0])

		assert.Contains(t, statements[0], `merge_maps(attributes, ParseJSON(attributes["metadata.dynatrace.com"]), "upsert")`)
		assert.Contains(t, statements[0], `IsMatch(attributes["metadata.dynatrace.com"], "^\\{")`)
		assert.Equal(t, `delete_key(attributes, "metadata.dynatrace.com")`, statements[1])
		assert.Equal(t, deleteOwnerFacts(false), statements[len(statements)-len(deleteOwnerFacts(false)):])
	})

	t.Run("resource statements are between the workload derivation and the deletion of the facts", func(t *testing.T) {
		cfg := NewBuilder().K8sEnrichment(K8sEnrichmentOpts{ResourceStatements: []string{"extra"}}).Build()

		statements := statementsOf(t, transformGroups(t, cfg, "metric_statements")[0])

		expected := append(append(mergeDynatraceAnnotations(), workloadFromOwners(false)...), "extra")
		assert.Equal(t, append(expected, deleteOwnerFacts(false)...), statements)
	})

	t.Run("extra groups follow the resource group", func(t *testing.T) {
		extra := map[string]any{"context": "datapoint", "statements": []string{"x"}}
		cfg := NewBuilder().K8sEnrichment(K8sEnrichmentOpts{Groups: []map[string]any{extra}}).Build()

		groups := transformGroups(t, cfg, "metric_statements")

		require.Len(t, groups, 2)
		assert.Equal(t, "resource", groups[0]["context"])
		assert.Equal(t, extra, groups[1])
	})
}

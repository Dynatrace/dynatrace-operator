// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package otelcgen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfigWithProcessors(t *testing.T) {
	cfg, err := NewConfig(
		"",
		RegisteredProtocols,
		WithProcessors(),
	)
	require.NoError(t, err)
	c, err := cfg.Marshal()
	require.NoError(t, err)

	expectedOutput, err := os.ReadFile(filepath.Join("testdata", "processors_only.yaml"))
	require.NoError(t, err)

	assert.YAMLEq(t, string(expectedOutput), string(c))
}

func TestK8sAttributesJSONAnnotation(t *testing.T) {
	cfg := &Config{}
	processors := cfg.buildProcessors()

	k8sAttr, ok := processors[k8sattributes].(map[string]any)
	require.True(t, ok)

	extract, ok := k8sAttr["extract"].(map[string]any)
	require.True(t, ok)

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

func TestDynatraceTransformationsJSONAnnotation(t *testing.T) {
	cfg := &Config{}
	transformations := cfg.dynatraceTransformations()
	require.Len(t, transformations, 1)

	statements, ok := transformations[0]["statements"].([]string)
	require.True(t, ok)
	require.GreaterOrEqual(t, len(statements), 2)

	t.Run("merge_maps statement is first", func(t *testing.T) {
		assert.Contains(t, statements[0], "merge_maps(attributes, ParseJSON(attributes[\"metadata.dynatrace.com\"]), \"insert\")")
		assert.Contains(t, statements[0], `IsMatch(attributes["metadata.dynatrace.com"], "^\\{")`)
	})

	t.Run("delete_key statement is second", func(t *testing.T) {
		assert.Equal(t, `delete_key(attributes, "metadata.dynatrace.com")`, statements[1])
	})

	t.Run("all signal types carry JSON annotation statements", func(t *testing.T) {
		cfg2 := &Config{}
		transform := cfg2.buildTransform()

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

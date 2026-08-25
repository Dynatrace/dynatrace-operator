// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package gateway

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildResourceAttributeStatements(t *testing.T) {
	t.Run("empty attrs", func(t *testing.T) {
		statements := buildResourceAttributeStatements(map[string]string{})
		assert.Empty(t, statements)
	})

	t.Run("single entry", func(t *testing.T) {
		statements := buildResourceAttributeStatements(map[string]string{"region": "us-east"})
		require.Len(t, statements, 1)
		assert.Equal(t,
			`set(attributes["region"], "us-east") where attributes["region"] == nil`,
			statements[0],
		)
	})

	t.Run("multiple entries sorted by key", func(t *testing.T) {
		statements := buildResourceAttributeStatements(map[string]string{
			"region":  "us-east",
			"dt.env":  "prod",
			"cluster": "main",
		})
		require.Len(t, statements, 3)
		assert.Equal(t, `set(attributes["cluster"], "main") where attributes["cluster"] == nil`, statements[0])
		assert.Equal(t, `set(attributes["dt.env"], "prod") where attributes["dt.env"] == nil`, statements[1])
		assert.Equal(t, `set(attributes["region"], "us-east") where attributes["region"] == nil`, statements[2])
	})

	t.Run("quotes in key and value are escaped", func(t *testing.T) {
		statements := buildResourceAttributeStatements(map[string]string{`k"ey`: `val"ue`})
		require.Len(t, statements, 1)
		assert.Equal(t,
			`set(attributes["k\"ey"], "val\"ue") where attributes["k\"ey"] == nil`,
			statements[0],
		)
	})

	t.Run("backslashes in key and value are escaped", func(t *testing.T) {
		statements := buildResourceAttributeStatements(map[string]string{`k\ey`: `val\ue`})
		require.Len(t, statements, 1)
		assert.Equal(t,
			`set(attributes["k\\ey"], "val\\ue") where attributes["k\\ey"] == nil`,
			statements[0],
		)
	})
}

func TestRenderGatewayConfig_ResourceAttributes(t *testing.T) {
	t.Run("empty resource attributes produces no baseline set block", func(t *testing.T) {
		data := gatewayConfigData{Endpoint: "https://example.com/v2/otlp"}
		rendered, err := renderGatewayConfig(data)
		require.NoError(t, err)
		// "== nil" only appears in the resource-attributes block; none of the static blocks use it.
		assert.NotContains(t, rendered, "== nil")
	})

	t.Run("resource attributes rendered as sorted baseline set statements", func(t *testing.T) {
		data := gatewayConfigData{
			Endpoint: "https://example.com/v2/otlp",
			ResourceAttributes: map[string]string{
				"region":            "us-east",
				"dt.k8s.cluster.id": "abc123",
			},
		}
		rendered, err := renderGatewayConfig(data)
		require.NoError(t, err)
		assert.Contains(t, rendered, `set(attributes["dt.k8s.cluster.id"], "abc123") where attributes["dt.k8s.cluster.id"] == nil`)
		assert.Contains(t, rendered, `set(attributes["region"], "us-east") where attributes["region"] == nil`)
	})

	t.Run("resource attributes block appears before annotation merge block", func(t *testing.T) {
		data := gatewayConfigData{
			Endpoint:           "https://example.com/v2/otlp",
			ResourceAttributes: map[string]string{"region": "us-east"},
		}
		rendered, err := renderGatewayConfig(data)
		require.NoError(t, err)

		setIdx := strings.Index(rendered, `set(attributes["region"], "us-east")`)
		mergeIdx := strings.Index(rendered, "merge_maps(")
		assert.Greater(t, mergeIdx, setIdx, "resource attributes block must precede the annotation merge block")
	})
}

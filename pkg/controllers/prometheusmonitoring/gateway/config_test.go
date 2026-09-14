// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package gateway

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

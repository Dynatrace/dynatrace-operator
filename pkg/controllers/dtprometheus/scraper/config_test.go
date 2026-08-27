// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package scraper

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildScraperConfigData(t *testing.T) {
	t.Run("endpoints are derived from the owner, not the scraper spec", func(t *testing.T) {
		s := newTestScope(newTestDTP("dtp", "dynatrace"))

		data := buildScraperConfigData(s)

		assert.Equal(t, "http://dtp-prometheus-allocator.dynatrace.svc.cluster.local:8080", data.TargetAllocatorEndpoint)
		assert.Equal(t, "dtp-gateway.dynatrace.svc.cluster.local:4317", data.GatewayEndpoint)
	})

	t.Run("endpoints follow the owner name and namespace", func(t *testing.T) {
		s := newTestScope(newTestDTP("other", "custom-ns"))

		data := buildScraperConfigData(s)

		assert.Equal(t, "http://other-prometheus-allocator.custom-ns.svc.cluster.local:8080", data.TargetAllocatorEndpoint)
		assert.Equal(t, "other-gateway.custom-ns.svc.cluster.local:4317", data.GatewayEndpoint)
	})

	t.Run("poll interval is taken from the spec", func(t *testing.T) {
		dtp := newTestDTP("dtp", "dynatrace")
		dtp.Spec.Scraper.TargetsPollInterval = metav1.Duration{Duration: 90 * time.Second}

		assert.Equal(t, "1m30s", buildScraperConfigData(newTestScope(dtp)).TargetsPollInterval)
	})
}

func TestBuildScraperOTelConfig(t *testing.T) {
	data := scraperConfigData{
		TargetAllocatorEndpoint: "http://ta.dynatrace.svc.cluster.local:8080",
		GatewayEndpoint:         "gw.dynatrace.svc.cluster.local:4317",
		TargetsPollInterval:     "1m0s",
	}

	t.Run("wires a single metrics pipeline", func(t *testing.T) {
		cfg := buildScraperOTelConfig(data)

		pipeline := cfg.Service.Pipelines[metricsSignalID]
		require.NotNil(t, pipeline)
		assert.Equal(t, []component.ID{prometheusID}, pipeline.Receivers)
		assert.Equal(t, []component.ID{memoryLimiterID}, pipeline.Processors)
		assert.Equal(t, []component.ID{otlpID}, pipeline.Exporters)
	})

	t.Run("receiver polls the target allocator", func(t *testing.T) {
		receiver, ok := buildScraperOTelConfig(data).Receivers[prometheusID].(map[string]any)
		require.True(t, ok)

		ta, ok := receiver["target_allocator"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, data.TargetAllocatorEndpoint, ta["endpoint"])
		assert.Equal(t, "1m0s", ta["interval"])
		// The target allocator assigns targets per collector, so each pod must identify itself.
		assert.Equal(t, "${env:MY_POD_NAME}", ta["collector_id"])
	})

	t.Run("exporter targets the gateway without TLS", func(t *testing.T) {
		exporter, ok := buildScraperOTelConfig(data).Exporters[otlpID].(map[string]any)
		require.True(t, ok)

		assert.Equal(t, data.GatewayEndpoint, exporter["endpoint"])
		assert.Equal(t, map[string]any{"insecure": true}, exporter["tls"])
	})

	t.Run("marshals to valid yaml", func(t *testing.T) {
		rendered, err := renderScraperConfig(data)

		require.NoError(t, err)
		assert.Contains(t, rendered, "prometheus:")
		assert.Contains(t, rendered, data.TargetAllocatorEndpoint)
		assert.Contains(t, rendered, data.GatewayEndpoint)
	})
}

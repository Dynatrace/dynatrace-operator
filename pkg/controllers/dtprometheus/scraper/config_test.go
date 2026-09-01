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

		assert.Equal(t, "http://dtp-allocator.dynatrace.svc.cluster.local:80", data.TargetAllocatorEndpoint)
		assert.Equal(t, "dtp-gateway.dynatrace", data.GatewayService)
	})

	t.Run("endpoints follow the owner name and namespace", func(t *testing.T) {
		s := newTestScope(newTestDTP("other", "custom-ns"))

		data := buildScraperConfigData(s)

		assert.Equal(t, "http://other-allocator.custom-ns.svc.cluster.local:80", data.TargetAllocatorEndpoint)
		assert.Equal(t, "other-gateway.custom-ns", data.GatewayService)
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
		GatewayService:          "gw.dynatrace",
		TargetsPollInterval:     "1m0s",
	}

	t.Run("wires a single metrics pipeline", func(t *testing.T) {
		cfg := buildScraperOTelConfig(data)

		pipeline := cfg.Service.Pipelines[metricsSignalID]
		require.NotNil(t, pipeline)
		assert.Equal(t, []component.ID{prometheusID}, pipeline.Receivers)
		assert.Equal(t, []component.ID{memoryLimiterID}, pipeline.Processors)
		assert.Equal(t, []component.ID{loadBalancingID}, pipeline.Exporters)
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

	t.Run("load-balancing exporter uses k8s resolver on the gateway service without TLS", func(t *testing.T) {
		exporter, ok := buildScraperOTelConfig(data).Exporters[loadBalancingID].(map[string]any)
		require.True(t, ok)

		resolver, ok := exporter["resolver"].(map[string]any)
		require.True(t, ok)

		k8s, ok := resolver["k8s"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, data.GatewayService, k8s["service"])
		assert.Equal(t, []int{gatewayOTLPPort}, k8s["ports"])

		protocol, ok := exporter["protocol"].(map[string]any)
		require.True(t, ok)

		otlp, ok := protocol["otlp"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, map[string]any{"insecure": true}, otlp["tls"])
	})

	t.Run("marshals to valid yaml", func(t *testing.T) {
		rendered, err := renderScraperConfig(data)

		require.NoError(t, err)
		assert.Contains(t, rendered, "prometheus:")
		assert.Contains(t, rendered, data.TargetAllocatorEndpoint)
		assert.Contains(t, rendered, "load_balancing:")
		assert.Contains(t, rendered, data.GatewayService)
	})
}

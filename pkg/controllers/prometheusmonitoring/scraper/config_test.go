// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package scraper

import (
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/otelcgen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pipeline"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildScraperConfigData(t *testing.T) {
	pm := newTestPM("other", "custom-ns")
	pm.Spec.Scraper.TargetsPollInterval = new(metav1.Duration{Duration: 90 * time.Second})

	data := buildScraperConfigData(newTestScope(pm))

	assert.Equal(t, "http://other-allocator.custom-ns:80", data.TargetAllocatorEndpoint)
	assert.Equal(t, "other-gateway.custom-ns", data.GatewayService)
	assert.Equal(t, "1m30s", data.TargetsPollInterval)
}

func Test_buildConfig(t *testing.T) {
	data := scraperConfigData{
		TargetAllocatorEndpoint: "http://ta.dynatrace:8080",
		GatewayService:          "gw.dynatrace",
		TargetsPollInterval:     "1m0s",
	}

	t.Run("wires a single metrics pipeline", func(t *testing.T) {
		cfg := buildConfig(data)

		p := cfg.Service.Pipelines[pipeline.NewID(pipeline.SignalMetrics)]
		require.NotNil(t, p)
		assert.Equal(t, []component.ID{prometheusID}, p.Receivers)
		assert.Equal(t, []component.ID{otelcgen.MemoryLimiterID}, p.Processors)
		assert.Equal(t, []component.ID{loadBalancingID}, p.Exporters)
	})

	t.Run("receiver polls the target allocator", func(t *testing.T) {
		cfg := buildConfig(data)

		receiver, ok := cfg.Receivers[prometheusID].(map[string]any)
		require.True(t, ok)

		ta, ok := receiver["target_allocator"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, data.TargetAllocatorEndpoint, ta["endpoint"])
		assert.Equal(t, "1m0s", ta["interval"])
		// The target allocator assigns targets per collector, so each pod must identify itself.
		assert.Equal(t, "${env:MY_POD_NAME}", ta["collector_id"])
	})

	t.Run("load-balancing exporter uses k8s resolver on the gateway service without TLS", func(t *testing.T) {
		cfg := buildConfig(data)

		exporter, ok := cfg.Exporters[loadBalancingID].(map[string]any)
		require.True(t, ok)

		resolver, ok := exporter["resolver"].(map[string]any)
		require.True(t, ok)

		k8s, ok := resolver["k8s"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, data.GatewayService, k8s["service"])
		assert.Equal(t, []int{otelcgen.OTLPReceiverGRPCPort}, k8s["ports"])

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

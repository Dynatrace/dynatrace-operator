package activegate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSpec_IsMode(t *testing.T) {
	tests := []struct {
		name         string
		capabilities []CapabilityDisplayName
		mode         CapabilityDisplayName
		expect       bool
	}{
		{"empty", nil, RoutingCapability.DisplayName, false},
		{"no match", []CapabilityDisplayName{KubeMonCapability.DisplayName, DynatraceAPICapability.DisplayName}, RoutingCapability.DisplayName, false},
		{"match", []CapabilityDisplayName{KubeMonCapability.DisplayName, RoutingCapability.DisplayName}, RoutingCapability.DisplayName, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ag := &Spec{Capabilities: tt.capabilities}
			assert.Equal(t, tt.expect, ag.IsMode(tt.mode))
		})
	}
}

func TestSpec_IsModeHelpers(t *testing.T) {
	ag := &Spec{}

	tests := []struct {
		name         string
		capabilities []CapabilityDisplayName
		check        func() bool
		expect       bool
	}{
		{"kubemon enabled", []CapabilityDisplayName{KubeMonCapability.DisplayName}, ag.IsKubernetesMonitoringEnabled, true},
		{"kubemon disabled", []CapabilityDisplayName{RoutingCapability.DisplayName}, ag.IsKubernetesMonitoringEnabled, false},
		{"routing enabled", []CapabilityDisplayName{RoutingCapability.DisplayName}, ag.IsRoutingEnabled, true},
		{"routing disabled", []CapabilityDisplayName{KubeMonCapability.DisplayName}, ag.IsRoutingEnabled, false},
		{"dynatrace api enabled", []CapabilityDisplayName{DynatraceAPICapability.DisplayName}, ag.IsAPIEnabled, true},
		{"dynatrace api disabled", []CapabilityDisplayName{MetricsIngestCapability.DisplayName}, ag.IsAPIEnabled, false},
		{"metrics ingest enabled", []CapabilityDisplayName{MetricsIngestCapability.DisplayName}, ag.IsMetricsIngestEnabled, true},
		{"metrics ingest disabled", []CapabilityDisplayName{DynatraceAPICapability.DisplayName}, ag.IsMetricsIngestEnabled, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ag.Capabilities = tt.capabilities
			assert.Equal(t, tt.expect, tt.check())
		})
	}
}

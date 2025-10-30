package exporter

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/otlp"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestTraceInjectorIsEnabledAndInject(t *testing.T) {
	apiURL := "http://example/api/v2/otlp"

	tests := []struct {
		name           string
		cfg            *otlp.ExporterConfiguration
		expectEnabled  bool
		expectInjected bool
		expectEnvVars  []string
	}{
		{
			name:           "nil config -> disabled",
			cfg:            nil,
			expectEnabled:  false,
			expectInjected: false,
		},
		{
			name:           "config without traces -> disabled",
			cfg:            &otlp.ExporterConfiguration{Spec: &otlp.ExporterConfigurationSpec{}},
			expectEnabled:  false,
			expectInjected: false,
		},
		{
			name:           "config with traces -> enabled and injects",
			cfg:            &otlp.ExporterConfiguration{Spec: &otlp.ExporterConfigurationSpec{Signals: otlp.SignalConfiguration{Traces: &otlp.TracesSignal{}}}},
			expectEnabled:  true,
			expectInjected: true,
			expectEnvVars:  []string{OTLPTraceEndpointEnv, OTLPTraceProtocolEnv},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inj := &traceInjector{cfg: tt.cfg}
			assert.Equal(t, tt.expectEnabled, inj.isEnabled())

			c := &corev1.Container{}
			injected := inj.Inject(c, apiURL, false)
			assert.Equal(t, tt.expectInjected, injected)

			for _, envName := range tt.expectEnvVars {
				assert.True(t, env.IsIn(c.Env, envName), "expected env var %s to be injected", envName)
			}
		})
	}
}

func TestMetricsInjectorIsEnabledAndInject(t *testing.T) {
	apiURL := "http://example/api/v2/otlp"

	tests := []struct {
		name           string
		cfg            *otlp.ExporterConfiguration
		expectEnabled  bool
		expectInjected bool
		expectEnvVars  []string
	}{
		{
			name:           "nil config -> disabled",
			cfg:            nil,
			expectEnabled:  false,
			expectInjected: false,
		},
		{
			name:           "config without metrics -> disabled",
			cfg:            &otlp.ExporterConfiguration{Spec: &otlp.ExporterConfigurationSpec{}},
			expectEnabled:  false,
			expectInjected: false,
		},
		{
			name:           "config with metrics -> enabled and injects",
			cfg:            &otlp.ExporterConfiguration{Spec: &otlp.ExporterConfigurationSpec{Signals: otlp.SignalConfiguration{Metrics: &otlp.MetricsSignal{}}}},
			expectEnabled:  true,
			expectInjected: true,
			expectEnvVars:  []string{OTLPMetricsEndpointEnv, OTLPMetricsProtocolEnv},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inj := &metricsInjector{cfg: tt.cfg}
			assert.Equal(t, tt.expectEnabled, inj.isEnabled())

			c := &corev1.Container{}
			injected := inj.Inject(c, apiURL, true) // override flag shouldn't matter for injection presence
			assert.Equal(t, tt.expectInjected, injected)

			for _, envName := range tt.expectEnvVars {
				assert.True(t, env.IsIn(c.Env, envName), "expected env var %s to be injected", envName)
			}
		})
	}
}

func TestLogsInjectorIsEnabledAndInject(t *testing.T) {
	apiURL := "http://example/api/v2/otlp"

	tests := []struct {
		name           string
		cfg            *otlp.ExporterConfiguration
		expectEnabled  bool
		expectInjected bool
		expectEnvVars  []string
	}{
		{
			name:           "nil config -> disabled",
			cfg:            nil,
			expectEnabled:  false,
			expectInjected: false,
		},
		{
			name:           "config without logs -> disabled",
			cfg:            &otlp.ExporterConfiguration{Spec: &otlp.ExporterConfigurationSpec{}},
			expectEnabled:  false,
			expectInjected: false,
		},
		{
			name:           "config with logs -> enabled and injects",
			cfg:            &otlp.ExporterConfiguration{Spec: &otlp.ExporterConfigurationSpec{Signals: otlp.SignalConfiguration{Logs: &otlp.LogsSignal{}}}},
			expectEnabled:  true,
			expectInjected: true,
			expectEnvVars:  []string{OTLPLogsEndpointEnv, OTLPLogsProtocolEnv},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inj := &logsInjector{cfg: tt.cfg}
			assert.Equal(t, tt.expectEnabled, inj.isEnabled())

			c := &corev1.Container{}
			injected := inj.Inject(c, apiURL, false)
			assert.Equal(t, tt.expectInjected, injected)

			for _, envName := range tt.expectEnvVars {
				assert.True(t, env.IsIn(c.Env, envName), "expected env var %s to be injected", envName)
			}
		})
	}
}

package exporter

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/otlpexporterconfiguration"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestTraceInjectorIsEnabledAndInject(t *testing.T) {
	apiURL := "http://example/api/v2/otlp"

	tests := []struct {
		name           string
		cfg            *otlpexporterconfiguration.OTLPExporterConfiguration
		addCertificate bool
		expectEnabled  bool
		expectInjected bool
		expectEnvVars  []string
	}{
		{
			name:           "nil config -> disabled",
			cfg:            nil,
			addCertificate: true, // even if requested, should not inject
			expectEnabled:  false,
			expectInjected: false,
		},
		{
			name:           "config without traces -> disabled",
			cfg:            &otlpexporterconfiguration.OTLPExporterConfiguration{Spec: &otlpexporterconfiguration.Spec{}},
			addCertificate: true,
			expectEnabled:  false,
			expectInjected: false,
		},
		{
			name:           "config with traces -> enabled and injects (no cert)",
			cfg:            &otlpexporterconfiguration.OTLPExporterConfiguration{Spec: &otlpexporterconfiguration.Spec{Signals: otlpexporterconfiguration.SignalConfiguration{Traces: &otlpexporterconfiguration.TracesSignal{}}}},
			addCertificate: false,
			expectEnabled:  true,
			expectInjected: true,
			expectEnvVars:  []string{OTLPTraceEndpointEnv, OTLPTraceProtocolEnv, OTLPTraceHeadersEnv},
		},
		{
			name:           "config with traces -> enabled and injects (with cert)",
			cfg:            &otlpexporterconfiguration.OTLPExporterConfiguration{Spec: &otlpexporterconfiguration.Spec{Signals: otlpexporterconfiguration.SignalConfiguration{Traces: &otlpexporterconfiguration.TracesSignal{}}}},
			addCertificate: true,
			expectEnabled:  true,
			expectInjected: true,
			expectEnvVars:  []string{OTLPTraceEndpointEnv, OTLPTraceProtocolEnv, OTLPTraceHeadersEnv, OTLPTraceCertificateEnv},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inj := &traceInjector{cfg: tt.cfg}
			assert.Equal(t, tt.expectEnabled, inj.isEnabled())

			c := &corev1.Container{}
			injected := inj.Inject(c, apiURL, tt.addCertificate)
			assert.Equal(t, tt.expectInjected, injected)

			assert.Len(t, c.Env, len(tt.expectEnvVars))

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
		cfg            *otlpexporterconfiguration.OTLPExporterConfiguration
		addCertificate bool
		expectEnabled  bool
		expectInjected bool
		expectEnvVars  []string
	}{
		{
			name:           "nil config -> disabled",
			cfg:            nil,
			addCertificate: true,
			expectEnabled:  false,
			expectInjected: false,
		},
		{
			name:           "config without metrics -> disabled",
			cfg:            &otlpexporterconfiguration.OTLPExporterConfiguration{Spec: &otlpexporterconfiguration.Spec{}},
			addCertificate: true,
			expectEnabled:  false,
			expectInjected: false,
		},
		{
			name:           "config with metrics -> enabled and injects (no cert)",
			cfg:            &otlpexporterconfiguration.OTLPExporterConfiguration{Spec: &otlpexporterconfiguration.Spec{Signals: otlpexporterconfiguration.SignalConfiguration{Metrics: &otlpexporterconfiguration.MetricsSignal{}}}},
			addCertificate: false,
			expectEnabled:  true,
			expectInjected: true,
			expectEnvVars:  []string{OTLPMetricsEndpointEnv, OTLPMetricsProtocolEnv, OTLPMetricsHeadersEnv},
		},
		{
			name:           "config with metrics -> enabled and injects (with cert)",
			cfg:            &otlpexporterconfiguration.OTLPExporterConfiguration{Spec: &otlpexporterconfiguration.Spec{Signals: otlpexporterconfiguration.SignalConfiguration{Metrics: &otlpexporterconfiguration.MetricsSignal{}}}},
			addCertificate: true,
			expectEnabled:  true,
			expectInjected: true,
			expectEnvVars:  []string{OTLPMetricsEndpointEnv, OTLPMetricsProtocolEnv, OTLPMetricsHeadersEnv, OTLPMetricsCertificateEnv},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inj := &metricsInjector{cfg: tt.cfg}
			assert.Equal(t, tt.expectEnabled, inj.isEnabled())

			c := &corev1.Container{}
			injected := inj.Inject(c, apiURL, tt.addCertificate) // override flag shouldn't matter for injection presence
			assert.Equal(t, tt.expectInjected, injected)

			assert.Len(t, c.Env, len(tt.expectEnvVars))

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
		cfg            *otlpexporterconfiguration.OTLPExporterConfiguration
		addCertificate bool
		expectEnabled  bool
		expectInjected bool
		expectEnvVars  []string
	}{
		{
			name:           "nil config -> disabled",
			cfg:            nil,
			addCertificate: true,
			expectEnabled:  false,
			expectInjected: false,
		},
		{
			name:           "config without logs -> disabled",
			cfg:            &otlpexporterconfiguration.OTLPExporterConfiguration{Spec: &otlpexporterconfiguration.Spec{}},
			addCertificate: true,
			expectEnabled:  false,
			expectInjected: false,
		},
		{
			name:           "config with logs -> enabled and injects (no cert)",
			cfg:            &otlpexporterconfiguration.OTLPExporterConfiguration{Spec: &otlpexporterconfiguration.Spec{Signals: otlpexporterconfiguration.SignalConfiguration{Logs: &otlpexporterconfiguration.LogsSignal{}}}},
			addCertificate: false,
			expectEnabled:  true,
			expectInjected: true,
			expectEnvVars:  []string{OTLPLogsEndpointEnv, OTLPLogsProtocolEnv, OTLPLogsHeadersEnv},
		},
		{
			name:           "config with logs -> enabled and injects (with cert)",
			cfg:            &otlpexporterconfiguration.OTLPExporterConfiguration{Spec: &otlpexporterconfiguration.Spec{Signals: otlpexporterconfiguration.SignalConfiguration{Logs: &otlpexporterconfiguration.LogsSignal{}}}},
			addCertificate: true,
			expectEnabled:  true,
			expectInjected: true,
			expectEnvVars:  []string{OTLPLogsEndpointEnv, OTLPLogsProtocolEnv, OTLPLogsHeadersEnv, OTLPLogsCertificateEnv},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inj := &logsInjector{cfg: tt.cfg}
			assert.Equal(t, tt.expectEnabled, inj.isEnabled())

			c := &corev1.Container{}
			injected := inj.Inject(c, apiURL, tt.addCertificate)
			assert.Equal(t, tt.expectInjected, injected)

			assert.Len(t, c.Env, len(tt.expectEnvVars))

			for _, envName := range tt.expectEnvVars {
				assert.True(t, env.IsIn(c.Env, envName), "expected env var %s to be injected", envName)
			}
		})
	}
}

package exporter

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/otlp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestTraceInjectorIsEnabledAndInject(t *testing.T) {
	apiURL := "http://example/api/v2/otlp"

	tests := []struct {
		name           string
		cfg            *otlp.ExporterConfiguration
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
			cfg:            &otlp.ExporterConfiguration{Spec: &otlp.ExporterConfigurationSpec{}},
			addCertificate: true,
			expectEnabled:  false,
			expectInjected: false,
		},
		{
			name:           "config with traces -> enabled and injects (no cert)",
			cfg:            &otlp.ExporterConfiguration{Spec: &otlp.ExporterConfigurationSpec{Signals: otlp.SignalConfiguration{Traces: &otlp.TracesSignal{}}}},
			addCertificate: false,
			expectEnabled:  true,
			expectInjected: true,
			expectEnvVars:  []string{OTLPTraceEndpointEnv, OTLPTraceProtocolEnv, OTLPTraceHeadersEnv},
		},
		{
			name:           "config with traces -> enabled and injects (with cert)",
			cfg:            &otlp.ExporterConfiguration{Spec: &otlp.ExporterConfigurationSpec{Signals: otlp.SignalConfiguration{Traces: &otlp.TracesSignal{}}}},
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
		cfg            *otlp.ExporterConfiguration
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
			cfg:            &otlp.ExporterConfiguration{Spec: &otlp.ExporterConfigurationSpec{}},
			addCertificate: true,
			expectEnabled:  false,
			expectInjected: false,
		},
		{
			name:           "config with metrics -> enabled and injects (no cert)",
			cfg:            &otlp.ExporterConfiguration{Spec: &otlp.ExporterConfigurationSpec{Signals: otlp.SignalConfiguration{Metrics: &otlp.MetricsSignal{}}}},
			addCertificate: false,
			expectEnabled:  true,
			expectInjected: true,
			expectEnvVars:  []string{OTLPMetricsEndpointEnv, OTLPMetricsProtocolEnv, OTLPMetricsHeadersEnv},
		},
		{
			name:           "config with metrics -> enabled and injects (with cert)",
			cfg:            &otlp.ExporterConfiguration{Spec: &otlp.ExporterConfigurationSpec{Signals: otlp.SignalConfiguration{Metrics: &otlp.MetricsSignal{}}}},
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
		cfg            *otlp.ExporterConfiguration
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
			cfg:            &otlp.ExporterConfiguration{Spec: &otlp.ExporterConfigurationSpec{}},
			addCertificate: true,
			expectEnabled:  false,
			expectInjected: false,
		},
		{
			name:           "config with logs -> enabled and injects (no cert)",
			cfg:            &otlp.ExporterConfiguration{Spec: &otlp.ExporterConfigurationSpec{Signals: otlp.SignalConfiguration{Logs: &otlp.LogsSignal{}}}},
			addCertificate: false,
			expectEnabled:  true,
			expectInjected: true,
			expectEnvVars:  []string{OTLPLogsEndpointEnv, OTLPLogsProtocolEnv, OTLPLogsHeadersEnv},
		},
		{
			name:           "config with logs -> enabled and injects (with cert)",
			cfg:            &otlp.ExporterConfiguration{Spec: &otlp.ExporterConfigurationSpec{Signals: otlp.SignalConfiguration{Logs: &otlp.LogsSignal{}}}},
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

func TestNoProxyInjector_Inject(t *testing.T) {
	type args struct {
		containsNoProxy     bool
		activeGateEnabled   bool
		featureFlagDisabled bool
		hasProxy            bool
		noProxyValue        string
		alreadyContainsFQDN bool
	}

	const agFQDN = "dynakube-activegate.dynatrace"

	makeDynakube := func(activeGateEnabled, featureFlagDisabled, hasProxy bool) *dynakube.DynaKube {
		dk := &dynakube.DynaKube{
			ObjectMeta: v1.ObjectMeta{
				Name:      "dynakube",
				Namespace: "dynatrace",
			},
		}

		if activeGateEnabled {
			dk.Spec.ActiveGate = activegate.Spec{
				Capabilities: []activegate.CapabilityDisplayName{
					activegate.MetricsIngestCapability.DisplayName,
				},
			}
		}

		if featureFlagDisabled {
			dk.Annotations = map[string]string{
				exp.OTLPInjectionSetNoProxy: "false",
			}
		}

		if hasProxy {
			dk.Spec.Proxy = &value.Source{Value: "proxy"}
		}

		return dk
	}

	tests := []struct {
		name          string
		args          args
		expectMutated bool
		expectValue   string
	}{
		{
			name: "NO_PROXY not present",
			args: args{
				containsNoProxy:     false,
				activeGateEnabled:   true,
				hasProxy:            true,
				noProxyValue:        "",
				alreadyContainsFQDN: false,
			},
			expectMutated: true,
			expectValue:   agFQDN,
		},
		{
			name: "NO_PROXY present, empty",
			args: args{
				containsNoProxy:     true,
				activeGateEnabled:   true,
				hasProxy:            true,
				noProxyValue:        "",
				alreadyContainsFQDN: false,
			},
			expectMutated: true,
			expectValue:   agFQDN,
		},
		{
			name: "NO_PROXY present, value not containing FQDN",
			args: args{
				containsNoProxy:     true,
				activeGateEnabled:   true,
				hasProxy:            true,
				noProxyValue:        "foo,bar",
				alreadyContainsFQDN: false,
			},
			expectMutated: true,
			expectValue:   "foo,bar," + agFQDN,
		},
		{
			name: "NO_PROXY present, value already contains FQDN",
			args: args{
				containsNoProxy:     true,
				activeGateEnabled:   true,
				hasProxy:            true,
				noProxyValue:        agFQDN,
				alreadyContainsFQDN: true,
			},
			expectMutated: false,
			expectValue:   agFQDN,
		},
		{
			name: "feature flag disabled",
			args: args{
				activeGateEnabled:   true,
				featureFlagDisabled: true,
				hasProxy:            true,
				noProxyValue:        "foo",
				alreadyContainsFQDN: false,
			},
			expectMutated: false,
		},
		{
			name: "ActiveGate disabled",
			args: args{
				activeGateEnabled:   false,
				hasProxy:            true,
				noProxyValue:        "foo",
				alreadyContainsFQDN: false,
			},
			expectMutated: false,
		},
		{
			name: "no proxy configured in Dynakube",
			args: args{
				activeGateEnabled:   true,
				hasProxy:            false,
				noProxyValue:        "foo",
				alreadyContainsFQDN: false,
			},
			expectMutated: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dk := makeDynakube(tt.args.activeGateEnabled, tt.args.featureFlagDisabled, tt.expectMutated)

			inj := &noProxyInjector{dk: *dk}

			c := &corev1.Container{}

			if tt.args.containsNoProxy {
				c.Env = append(c.Env, corev1.EnvVar{Name: NoProxyEnv, Value: tt.args.noProxyValue})
			}

			mutated := inj.Inject(c, "", false)

			assert.Equal(t, tt.expectMutated, mutated)

			if tt.expectValue != "" {
				assert.Equal(t, tt.expectValue, env.FindEnvVar(c.Env, NoProxyEnv).Value)
			} else {
				assert.Nil(t, env.FindEnvVar(c.Env, NoProxyEnv))
			}
		})
	}
}

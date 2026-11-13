// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package otelcfg

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pipeline"
)

var (
	testOnlyID       = component.MustNewID("test_only")
	testOnlyID2      = component.MustNewID("test_only2")
	testReceiverOpts = ReceiverOpts{
		CertFile: "/run/opensignals/tls/tls.crt",
		KeyFile:  "/run/opensignals/tls/tls.key",
	}
)

func TestBuilder(t *testing.T) {
	tests := []struct {
		golden  string
		mutate  func(b *Builder)
		signals []pipeline.Signal
	}{
		{
			"exporter_apitoken",
			func(b *Builder) {
				b.OTLPHTTPExporter("test", ExporterOpts{
					CAFile:                   "/run/opensignals/cacerts/certs",
					APIToken:                 "test-token",
					IncludeSystemCACertsPool: true,
				})
			},
			nil,
		},
		{
			"exporter_bearertokenauth",
			func(b *Builder) {
				b.OTLPHTTPExporter("test", ExporterOpts{
					CAFile:          "/run/opensignals/cacerts/certs",
					BearerTokenAuth: true,
				})
			},
			nil,
		},
		{
			"extensions_only",
			func(b *Builder) { b.HealthCheck().BearerTokenAuth("test") },
			nil,
		},
		{
			"receivers_statsd",
			func(b *Builder) { b.ReceiverForProtocol(StatsdProtocol, ReceiverOpts{}) },
			nil,
		},
		{
			"receivers_zipkin_only",
			func(b *Builder) { b.ReceiverForProtocol(ZipkinProtocol, testReceiverOpts) },
			nil,
		},
		{
			"receivers_jaeger_only",
			func(b *Builder) { b.ReceiverForProtocol(JaegerProtocol, testReceiverOpts) },
			nil,
		},
		{
			"receivers_otlp_only",
			func(b *Builder) { b.ReceiverForProtocol(OTLPProtocol, testReceiverOpts) },
			nil,
		},
		{
			"receivers_otlp_grpc",
			func(b *Builder) { b.OTLPReceiver("grpc", OTLPReceiverGRPCPort).SelfMonitoring("") },
			[]pipeline.Signal{pipeline.SignalMetrics},
		},
		{
			"telemetry_metrics",
			func(b *Builder) { b.SelfMonitoring("test") },
			[]pipeline.Signal{pipeline.SignalMetrics},
		},
	}

	for _, test := range tests {
		t.Run(test.golden, func(t *testing.T) {
			b := NewBuilder()
			test.mutate(b)
			cfg := b.Build(test.signals...)

			got, err := cfg.Marshal()
			require.NoError(t, err)

			expect, err := os.ReadFile(filepath.Join("testdata", test.golden+".yaml"))
			require.NoError(t, err)

			assert.YAMLEqf(t, string(expect), string(got), "EXPECT:\n%s\nACTUAL:\n%s", string(expect), string(got))
		})
	}
}

func TestBuilder_Deduplicate(t *testing.T) {
	notEmpty := map[string]any{"foo": "bar"}

	cfg := NewBuilder().
		Receiver(testOnlyID, notEmpty).
		Receiver(testOnlyID, Empty).
		Processor(testOnlyID, notEmpty).
		Processor(testOnlyID, Empty).
		Exporter(testOnlyID, notEmpty).
		Exporter(testOnlyID, Empty).
		Build(pipeline.SignalMetrics)

	assert.Empty(t, cfg.Receivers[testOnlyID])
	assert.Empty(t, cfg.Processors[testOnlyID])
	assert.Empty(t, cfg.Exporters[testOnlyID])
	assert.NotEmpty(t, cfg.Service.Pipelines)
	for _, cfg := range cfg.Service.Pipelines {
		assert.Equal(t, []component.ID{testOnlyID}, cfg.Receivers)
		assert.Equal(t, []component.ID{testOnlyID}, cfg.Processors)
		assert.Equal(t, []component.ID{testOnlyID}, cfg.Receivers)
	}
}

func TestBuilder_BuildCopy(t *testing.T) {
	b := NewBuilder()
	emptyCfg := b.Build()
	assert.Empty(t, emptyCfg.Service.Pipelines)

	firstCfg := b.Processor(testOnlyID, Empty).Receiver(testOnlyID, Empty).Exporter(testOnlyID, Empty).Build()

	assert.NotEqual(t, emptyCfg, firstCfg)
	assert.Empty(t, emptyCfg.Service.Pipelines)

	secondCfg := b.Processor(testOnlyID2, Empty).Receiver(testOnlyID2, Empty).Exporter(testOnlyID, Empty).Build()
	assert.NotEqual(t, firstCfg, secondCfg)
	assert.NotEqual(t, firstCfg.Service.Pipelines, secondCfg.Service.Pipelines)
}

func TestBuilder_ReceiverForProtocol_Unknown(t *testing.T) {
	cfg := NewBuilder().ReceiverForProtocol("unknown", ReceiverOpts{
		CertFile: "/run/opensignals/tls/tls.crt",
		KeyFile:  "/run/opensignals/tls/tls.key",
	}).Build()
	assert.Empty(t, cfg.Receivers)
}

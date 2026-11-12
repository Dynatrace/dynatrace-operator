// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package otelcfg

import (
	"go.opentelemetry.io/collector/pipeline"
)

const (
	OTLPReceiverGRPCPort = 4317
	OTLPReceiverHTTPPort = 4318

	JaegerReceiverGRPCPort          = 14250
	JaegerReceiverThriftBinaryPort  = 6832
	JaegerReceiverThriftCompactPort = 6831
	JaegerReceiverThriftHTTPPort    = 14268

	ZipkinReceiverPort = 9411

	StatsdReceiverPort = 8125
)

type ReceiverOpts struct {
	CertFile string
	KeyFile  string
}

// https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.157.0/receiver/statsdreceiver/protocol/protocol.go#L47
type histogramConfig struct {
	MaxSize int32 `mapstructure:"max_size"`
}

// https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.157.0/receiver/statsdreceiver/protocol/protocol.go#L38
type timerHistogramMapping struct {
	StatsDType   string          `mapstructure:"statsd_type"`
	ObserverType string          `mapstructure:"observer_type"`
	Histogram    histogramConfig `mapstructure:"histogram"`
}

func (b *Builder) ReceiverForProtocol(protocol Protocol, opts ReceiverOpts) *Builder {
	switch protocol {
	case OTLPProtocol:
		return b.Receiver(OTLPID, map[string]any{"protocols": map[string]any{
			"grpc": buildReceiverConfig(OTLPReceiverGRPCPort, opts),
			"http": buildReceiverConfig(OTLPReceiverHTTPPort, opts),
		}})
	// zipkin https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.157.0/receiver/zipkinreceiver/factory.go#L25
	// jaeger https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.157.0/receiver/jaegerreceiver/factory.go#L31
	// only support traces.
	case JaegerProtocol:
		return b.Receiver(JaegerID, map[string]any{"protocols": map[string]any{
			"grpc":           buildReceiverConfig(JaegerReceiverGRPCPort, opts),
			"thrift_binary":  buildReceiverConfig(JaegerReceiverThriftBinaryPort, ReceiverOpts{}),
			"thrift_compact": buildReceiverConfig(JaegerReceiverThriftCompactPort, ReceiverOpts{}),
			"thrift_http":    buildReceiverConfig(JaegerReceiverThriftHTTPPort, opts),
		}}, pipeline.SignalTraces)
	case ZipkinProtocol:
		return b.Receiver(ZipkinID, buildReceiverConfig(ZipkinReceiverPort, opts), pipeline.SignalTraces)
	// statsd https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.157.0/receiver/statsdreceiver/factory.go#L31
	// only supports metrics.
	case StatsdProtocol:
		return b.Receiver(StatsdID, map[string]any{
			"endpoint": Endpoint(StatsdReceiverPort),
			"timer_histogram_mapping": []timerHistogramMapping{
				{
					StatsDType: "histogram", ObserverType: "histogram", Histogram: histogramConfig{MaxSize: 10},
				},
				{
					StatsDType: "timing", ObserverType: "histogram", Histogram: histogramConfig{MaxSize: 100},
				},
				{
					StatsDType: "distribution", ObserverType: "histogram", Histogram: histogramConfig{MaxSize: 100},
				},
			},
		}, pipeline.SignalMetrics)
	}

	return b
}

func buildReceiverConfig(port int, opts ReceiverOpts) map[string]any {
	// https://github.com/open-telemetry/opentelemetry-collector/blob/v0.157.0/config/confighttp/server.go#L38
	cfg := map[string]any{
		"endpoint": Endpoint(port),
	}

	if opts.CertFile != "" || opts.KeyFile != "" {
		tls := make(map[string]any)

		if opts.CertFile != "" {
			tls["cert_file"] = opts.CertFile
		}

		if opts.KeyFile != "" {
			tls["key_file"] = opts.KeyFile
		}

		cfg["tls"] = tls
	}

	return cfg
}

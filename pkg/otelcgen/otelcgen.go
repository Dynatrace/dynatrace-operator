package otelcgen

import (
	"fmt"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/otelcol"
	"go.opentelemetry.io/collector/receiver/otlpreceiver"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"gopkg.in/yaml.v3"
)

type Protocol string

type Protocols []Protocol

const (
	JagerProtocol  Protocol = "jager"
	ZipkinProtocol Protocol = "zipkin"
	OtlpProtocol   Protocol = "otlp"
	StatsdProtocol Protocol = "statsd"
)

type Option func(c *otelcol.Config)

func NewConfig(options ...Option) (*otelcol.Config, error) {
	c := otelcol.Config{}

	for _, opt := range options {
		opt(&c)
	}

	return &c, nil
}

func WithProtocols(protocols ...string) Option {
	if len(protocols) == 0 {
		// means all protocols
		protocols = []string{string(JagerProtocol)}
	}

	// TODO: dynamically and conditionally create all maps in a loop
	jagerID := component.MustNewID(string(JagerProtocol))

	jagerExporter, _ := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint("http://localhost:14268/api/traces")))
	exporters := map[component.ID]component.Config{
		jagerID: jagerExporter,
	}

	receivers := map[component.ID]component.Config{
		jagerID: otlpreceiver.Config{Protocols: otlpreceiver.Protocols{}},
	}
	return func(c *otelcol.Config) {
		c.Exporters = exporters
		c.Receivers = receivers
	}
}

func usage() {
	c, _ := NewConfig(WithProtocols())
	conf := confmap.New()
	_ = conf.Marshal(c)
	by, _ := yaml.Marshal(conf.ToStringMap())
	fmt.Println(string(by))
}

// Usage:
// c := &otelcgen.NewConfig(otelcgen.WithProtocols("jager", "zipkin"), otelcgen.WithService(), otelcgen.WithCustomTLS(""))
// conf := confmap.New()
// by, _ := yaml.Marshal(conf.ToStringMap())

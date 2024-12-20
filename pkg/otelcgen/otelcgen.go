package otelcgen

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/otelcol"
	"go.opentelemetry.io/collector/receiver/otlpreceiver"
	"sigs.k8s.io/yaml"
)

type Protocol string

type Protocols []Protocol

const (
	JagerProtocol  Protocol = "jager"
	ZipkinProtocol Protocol = "zipkin"
	OtlpProtocol   Protocol = "otlp"
	StatsdProtocol Protocol = "statsd"
)

type Config struct {
	cfg *otelcol.Config
}

func (c Config) Marshal() ([]byte, error) {
	conf := confmap.New()
	err := conf.Marshal(c.cfg)
	if err != nil {
		return nil, err
	}
	m := conf.ToStringMap()
	return yaml.Marshal(m)
}

type Option func(c *Config)

func NewConfig(options ...Option) (*Config, error) {
	c := Config{
		cfg: &otelcol.Config{},
	}

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
	//jagerID := component.MustNewID(string(JagerProtocol))
	otlpID := component.MustNewID(string(OtlpProtocol))

	//jagerExporter, _ := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint("http://localhost:14268/api/traces")))
	//exporters := map[component.ID]component.Config{}

	receivers := map[component.ID]component.Config{
		otlpID: otlpreceiver.Protocols{
			GRPC: &configgrpc.ServerConfig{
				NetAddr: confignet.AddrConfig{
					Endpoint: "test",
				},
				TLSSetting: &configtls.ServerConfig{
					Config: configtls.Config{
						CAFile:  "/run/opensignals/tls/tls.crt",
						KeyFile: "/run/opensignals/tls/tls.key",
					},
				},
			},
			HTTP: nil,
		},
	}

	return func(c *Config) {
		c.cfg.Receivers = receivers
	}
}

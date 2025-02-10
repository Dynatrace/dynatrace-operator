package otelcgen

import (
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/otelcol"
	"gopkg.in/yaml.v3"
)

// HistogramConfig is based on:
// "go.opentelemetry.io/opentelemetry-collector-contrib/receiver/statsdreceiver/internal/protocol/statsd_parser.go.HistogramConfig"
// with reduced number of attributes to reduce the number of dependencies.
type HistogramConfig struct {
	MaxSize int32 `mapstructure:"max_size"`
}

// TimerHistogramMapping is based on:
// "go.opentelemetry.io/opentelemetry-collector-contrib/receiver/statsdreceiver/internal/protocol/statsd_parser.go.TLSSetting"
// with reduced number of attributes to reduce the number of dependencies.
type TimerHistogramMapping struct {
	StatsDType   string          `mapstructure:"statsd_type"`
	ObserverType string          `mapstructure:"observer_type"`
	Histogram    HistogramConfig `mapstructure:"histogram"`
}

// TLSSetting is based on:
// "go.opentelemetry.io/collector/config/configtls.TLSSetting"
// with reduced number of attributes to reduce the number of dependencies.
type TLSSetting struct {
	CAFile   string `mapstructure:"ca_file,omitempty"`
	KeyFile  string `mapstructure:"key_file,omitempty"`
	CertFile string `mapstructure:"cert_file,omitempty"`
}

// ServerConfig is based on "go.opentelemetry.io/collector/config/confighttp.ServerConfig" and
// "go.opentelemetry.io/collector/config/confighttp.ServerConfig" with reduced number of attributes
// to reduce the number of dependencies.
type ServerConfig struct {
	// TLSSetting struct exposes TLS client configuration.
	TLSSetting *TLSSetting `mapstructure:"tls,omitempty"`

	// Additional headers attached to each HTTP request sent by the client.
	// Existing header values are overwritten if collision happens.
	// Header values are opaque since they may be sensitive.
	Headers map[string]string `mapstructure:"headers,omitempty"`

	// The target URL to send data to (e.g.: http://some.url:9411/v1/traces).
	Endpoint string `mapstructure:"endpoint"`
}

type Protocol string

type Protocols []Protocol

const (
	JaegerProtocol Protocol = "jaeger"
	ZipkinProtocol Protocol = "zipkin"
	OtlpProtocol   Protocol = "otlp"
	StatsdProtocol Protocol = "statsd"
)

var (
	JaegerID = component.MustNewID(string(JaegerProtocol))
	OtlpID   = component.MustNewID(string(OtlpProtocol))
	StatsdID = component.MustNewID(string(StatsdProtocol))
	ZipkinID = component.MustNewID(string(ZipkinProtocol))
)

type Config struct {
	cfg      *otelcol.Config
	tlsKey   string
	tlsCert  string
	caFile   string
	podIP    string
	apiToken string
}

type Option func(c *Config) error

func NewConfig(podIP string, options ...Option) (*Config, error) {
	c := Config{
		cfg:   &otelcol.Config{},
		podIP: podIP,
	}

	for _, opt := range options {
		if err := opt(&c); err != nil {
			return nil, err
		}
	}

	return &c, nil
}

func (c *Config) Marshal() ([]byte, error) {
	conf := confmap.New()
	err := conf.Marshal(c.cfg)

	if err != nil {
		return nil, err
	}

	return yaml.Marshal(conf.ToStringMap())
}

func (c *Config) buildTLSSetting() *TLSSetting {
	tls := &TLSSetting{}
	if c.tlsCert != "" {
		tls.CertFile = c.tlsCert
	}

	if c.tlsKey != "" {
		tls.KeyFile = c.tlsKey
	}

	return tls
}

func (c *Config) buildEndpoint(port uint) string {
	return fmt.Sprintf("%s:%d", c.podIP, port)
}

func (c *Config) buildEndpointWithoutPort() string {
	return c.podIP
}

func WithProtocols(protocols ...string) Option {
	return func(c *Config) error {
		receivers, err := c.buildReceivers(protocols)
		if err != nil {
			return err
		}

		c.cfg.Receivers = receivers

		return nil
	}
}

func WithProcessors() Option {
	return func(c *Config) error {
		processors := c.buildProcessors()

		c.cfg.Processors = processors

		return nil
	}
}

func WithExporters() Option {
	return func(c *Config) error {
		exporters := c.buildExporters()

		c.cfg.Exporters = exporters

		return nil
	}
}

func WithExtensions() Option {
	return func(c *Config) error {
		extensions := c.buildExtensions()

		c.cfg.Extensions = extensions

		return nil
	}
}

func WithServices() Option {
	return func(c *Config) error {
		services := c.buildServices()

		c.cfg.Service = services

		return nil
	}
}

func WithTLS(tlsCert, tlsKey string) Option {
	return func(c *Config) error {
		c.tlsCert = tlsCert
		c.tlsKey = tlsKey

		return nil
	}
}

func WithCA(caFile string) Option {
	return func(c *Config) error {
		c.caFile = caFile

		return nil
	}
}

func WithApiToken(apiToken string) Option {
	return func(c *Config) error {
		c.apiToken = apiToken

		return nil
	}
}

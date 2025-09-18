package otelcgen

import (
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/confmap"
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

type TLSSetting = configtls.ServerConfig

// ServerConfig is based on "go.opentelemetry.io/collector/config/confighttp.ServerConfig"
// with reduced number of attributes  to reduce the number of dependencies.
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

type DebugExporter struct {
	Verbosity          string `mapstructure:"verbosity"`
	SamplingInitial    int    `mapstructure:"sampling_initial"`
	SamplingThereafter int    `mapstructure:"sampling_thereafter"`
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

	RegisteredProtocols = Protocols{OtlpProtocol, JaegerProtocol, StatsdProtocol, ZipkinProtocol}
)

type Config struct {
	// Receivers is a map of ComponentID to Receivers.
	Receivers map[component.ID]component.Config `mapstructure:"receivers"`

	// Exporters is a map of ComponentID to Exporters.
	Exporters map[component.ID]component.Config `mapstructure:"exporters"`

	// Processors is a map of ComponentID to Processors.
	Processors map[component.ID]component.Config `mapstructure:"processors"`

	// Connectors is a map of ComponentID to connectors.
	Connectors map[component.ID]component.Config `mapstructure:"connectors"`

	// Extensions is a map of ComponentID to extensions.
	Extensions map[component.ID]component.Config `mapstructure:"extensions"`

	tlsKey   string
	tlsCert  string
	caFile   string
	podIP    string
	endpoint string
	apiToken string

	Service   ServiceConfig `mapstructure:"service"`
	protocols Protocols

	includeSystemCACertsPool bool

	debugExporterVerbosity          string
	debugExporterSamplingInitial    int
	debugExporterSamplingThereafter int
}

type Option func(c *Config) error

func NewConfig(podIP string, protocols Protocols, options ...Option) (*Config, error) {
	c := Config{
		podIP:     podIP,
		protocols: protocols,
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

	if err := conf.Marshal(c); err != nil {
		return nil, err
	}

	sm := conf.ToStringMap()

	return yaml.Marshal(sm)
}

func (c *Config) buildTLSSetting() *TLSSetting {
	if c.tlsCert == "" && c.tlsKey == "" {
		return nil
	}

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

func (c *Config) buildExportersEndpoint() string {
	return c.endpoint
}

func (c *Config) protocolsToIDs() []component.ID {
	ids := []component.ID{}

	for _, p := range c.protocols {
		switch p {
		case JaegerProtocol:
			ids = append(ids, JaegerID)
		case ZipkinProtocol:
			ids = append(ids, ZipkinID)
		case StatsdProtocol:
			ids = append(ids, StatsdID)
		case OtlpProtocol:
			ids = append(ids, OtlpID)
		}
	}

	return ids
}

func WithReceivers() Option {
	return func(c *Config) error {
		receivers, err := c.buildReceivers()
		if err != nil {
			return err
		}

		c.Receivers = receivers

		return nil
	}
}

func WithProcessors() Option {
	return func(c *Config) error {
		processors := c.buildProcessors()

		c.Processors = processors

		return nil
	}
}

func WithExporters() Option {
	return func(c *Config) error {
		exporters := c.buildExporters()

		c.Exporters = exporters

		return nil
	}
}

func WithDebugExporter(verbosity string, samplingInitial, samplingThereafter int) Option {
	return func(c *Config) error {
		c.debugExporterVerbosity = verbosity
		c.debugExporterSamplingInitial = samplingInitial
		c.debugExporterSamplingThereafter = samplingThereafter

		return nil
	}
}

func WithExtensions() Option {
	return func(c *Config) error {
		extensions := c.buildExtensions()

		c.Extensions = extensions

		return nil
	}
}

func WithServices() Option {
	return func(c *Config) error {
		services := c.buildServices()

		c.Service = services

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

func WithSystemCAs(useSystemCAs bool) Option {
	return func(c *Config) error {
		c.includeSystemCACertsPool = useSystemCAs

		return nil
	}
}

func WithAPIToken(apiToken string) Option {
	return func(c *Config) error {
		c.apiToken = apiToken

		return nil
	}
}

func WithExportersEndpoint(endpoint string) Option {
	return func(c *Config) error {
		c.endpoint = endpoint

		return nil
	}
}

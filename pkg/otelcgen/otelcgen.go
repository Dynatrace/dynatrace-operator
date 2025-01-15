package otelcgen

import (
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/otelcol"
	"gopkg.in/yaml.v3"
)

type Histogram struct {
	MaxSize int32 `mapstructure:"max_size"`
}

type TimerHistogramMapping struct {
	StatsDType   string    `mapstructure:"statsd_type"`
	ObserverType string    `mapstructure:"observer_type"`
	Histogram    Histogram `mapstructure:"histogram"`
}

// TLSSetting is based on:
// "go.opentelemetry.io/collector/config/configtls.TLSSetting"
// with reduced number of attributes to reduce the number of dependencies.
type TLSSetting struct {
	CAFile   string `mapstructure:"ca_file,omitempty"`
	KeyFile  string `mapstructure:"key_file,omitempty"`
	CertFile string `mapstructure:"cert_file,omitempty"`
}

// ServerConfig is based on "go.opentelemetry.io/collector/config/configgrpc.ServerConfig" and
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

	StatsdDefaultEndpoint = "test"
	ZipkinDefaultEndpoint = "test"
)

var (
	JaegerID = component.MustNewID(string(JaegerProtocol))
	OtlpID   = component.MustNewID(string(OtlpProtocol))
	StatsdID = component.MustNewID(string(StatsdProtocol))
	ZipkinID = component.MustNewID(string(ZipkinProtocol))
)

type Config struct {
	cfg     *otelcol.Config
	tlsKey  string
	tlsCert string
}

type Option func(c *Config) error

func NewConfig(options ...Option) (*Config, error) {
	c := Config{
		cfg: &otelcol.Config{},
	}

	for _, opt := range options {
		err := opt(&c)
		if err != nil {
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

// func
// receivers
func (c *Config) buildReceiverComponent(componentID component.ID) component.Config {
	switch componentID {
	case OtlpID:
		return map[string]any{"protocols": map[string]any{
			"grpc": &ServerConfig{TLSSetting: c.buildTLSSetting(), Endpoint: "test:4317"},
			"http": &ServerConfig{TLSSetting: c.buildTLSSetting(), Endpoint: "test:4318"},
		}}
	case JaegerID:
		return map[string]any{"protocols": map[string]any{
			"grpc":           &ServerConfig{Endpoint: "test", TLSSetting: c.buildTLSSetting()},
			"thrift_binary":  &ServerConfig{Endpoint: "test:6832"},
			"thrift_compact": &ServerConfig{Endpoint: "test:6831"},
			"thrift_http":    &ServerConfig{Endpoint: "test:14268", TLSSetting: c.buildTLSSetting()},
		}}
	case ZipkinID:
		return &ServerConfig{
			Endpoint:   "test",
			TLSSetting: c.buildTLSSetting(),
		}
	case StatsdID:
		return map[string]any{
			"endpoint": "test",
			"timer_histogram_mapping": []TimerHistogramMapping{{
				StatsDType: "histogram", ObserverType: "histogram", Histogram: Histogram{MaxSize: 10},
			}},
		}
	}

	return nil
}

func (c *Config) buildReceivers(protocols []string) (map[component.ID]component.Config, error) {
	if len(protocols) == 0 {
		// means all protocols are enabled
		protocols = []string{string(StatsdProtocol), string(ZipkinProtocol), string(JaegerProtocol), string(OtlpProtocol)}
	}

	receivers := make(map[component.ID]component.Config)

	for _, p := range protocols {
		switch Protocol(p) {
		case StatsdProtocol:
			receivers[StatsdID] = c.buildReceiverComponent(StatsdID)
		case ZipkinProtocol:
			receivers[ZipkinID] = c.buildReceiverComponent(ZipkinID)
		case JaegerProtocol:
			receivers[JaegerID] = c.buildReceiverComponent(JaegerID)
		case OtlpProtocol:
			receivers[OtlpID] = c.buildReceiverComponent(OtlpID)
		default:
			return nil, fmt.Errorf("unknown protocol: %s", p)
		}
	}

	return receivers, nil
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

func WithTLSKey(tlsKey string) Option {
	return func(c *Config) error {
		c.tlsKey = tlsKey

		return nil
	}
}

func WithTLSCert(tlsCert string) Option {
	return func(c *Config) error {
		c.tlsCert = tlsCert

		return nil
	}
}

package otelcgen

import (
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
	KeyFile  string `mapstructure:"key_file"`
	CertFile string `mapstructure:"cert_file"`
}

// ServerConfig is based on "go.opentelemetry.io/collector/config/configgrpc.ServerConfig" and
// "go.opentelemetry.io/collector/config/confighttp.ServerConfig" with reduced number of attributes
// to reduce the number of dependencies.
type ServerConfig struct {
	TLSSetting *TLSSetting `mapstructure:"tls,omitempty"`
	Endpoint   string      `mapstructure:"endpoint"`
}

type Protocol string

type Protocols []Protocol

const (
	JagerProtocol  Protocol = "jager"
	ZipkinProtocol Protocol = "zipkin"
	OtlpProtocol   Protocol = "otlp"
	StatsdProtocol Protocol = "statsd"

	StatsdDefaultEndpoint = "test"
	ZipkinDefaultEndpoint = "test"
)

var (
	JagerID  = component.MustNewID(string(JagerProtocol))
	OtlpID   = component.MustNewID(string(OtlpProtocol))
	StatsdID = component.MustNewID(string(StatsdProtocol))
	ZipkinID = component.MustNewID(string(ZipkinProtocol))
)

type Config struct {
	cfg     *otelcol.Config
	tlsKey  string
	tlsCert string
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
	case JagerID:
		return map[string]any{"protocols": map[string]any{"grpc": &ServerConfig{
			Endpoint:   "test",
			TLSSetting: c.buildTLSSetting(),
		}}}
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

func (c *Config) buildReceivers(protocols []string) map[component.ID]component.Config {
	if len(protocols) == 0 {
		// means all protocols
		protocols = []string{string(StatsdProtocol), string(ZipkinProtocol), string(JagerProtocol), string(OtlpProtocol)}
	}

	receivers := make(map[component.ID]component.Config)

	for _, p := range protocols {
		switch Protocol(p) {
		case StatsdProtocol:
			receivers[StatsdID] = c.buildReceiverComponent(StatsdID)
		case ZipkinProtocol:
			receivers[ZipkinID] = c.buildReceiverComponent(ZipkinID)
		case JagerProtocol:
			receivers[JagerID] = c.buildReceiverComponent(JagerID)
		case OtlpProtocol:
			receivers[OtlpID] = c.buildReceiverComponent(OtlpID)
		}
	}

	return receivers
}

func WithProtocols(protocols ...string) Option {
	return func(c *Config) {
		c.cfg.Receivers = c.buildReceivers(protocols)
	}
}

func WithTLSKey(tlsKey string) Option {
	return func(c *Config) {
		c.tlsKey = tlsKey
	}
}

func WithTLSCert(tlsCert string) Option {
	return func(c *Config) {
		c.tlsCert = tlsCert
	}
}

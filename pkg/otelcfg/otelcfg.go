// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package otelcfg

import (
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/service/extensions"
	"go.opentelemetry.io/collector/service/pipelines"
	"gopkg.in/yaml.v3"
)

// Empty is an empty config value. It instructs OTEL to use the defaults for that components.
var Empty component.Config = make(map[string]any)

type Protocol string

const (
	JaegerProtocol Protocol = "jaeger"
	ZipkinProtocol Protocol = "zipkin"
	OTLPProtocol   Protocol = "otlp"
	StatsdProtocol Protocol = "statsd"
)

var (
	JaegerID = component.MustNewID(string(JaegerProtocol))
	OTLPID   = component.MustNewID(string(OTLPProtocol))
	StatsdID = component.MustNewID(string(StatsdProtocol))
	ZipkinID = component.MustNewID(string(ZipkinProtocol))

	RegisteredProtocols = []Protocol{OTLPProtocol, JaegerProtocol, StatsdProtocol, ZipkinProtocol}
)

// https://github.com/open-telemetry/opentelemetry-collector/blob/v0.157.0/otelcol/unmarshaler.go#L21
type Config struct {
	Receivers  map[component.ID]component.Config `mapstructure:"receivers"`
	Exporters  map[component.ID]component.Config `mapstructure:"exporters"`
	Processors map[component.ID]component.Config `mapstructure:"processors"`
	Extensions map[component.ID]component.Config `mapstructure:"extensions"`
	Service    ServiceConfig                     `mapstructure:"service"`
}

// https://github.com/open-telemetry/opentelemetry-collector/blob/v0.157.0/service/config.go#L13
type ServiceConfig struct {
	// Extensions are the ordered list of extensions configured for the service.
	Extensions extensions.Config `mapstructure:"extensions,omitempty"`

	// Pipelines are the set of data pipelines configured for the service.
	Pipelines pipelines.Config `mapstructure:"pipelines"`
}

func (c *Config) Marshal() ([]byte, error) {
	conf := confmap.New()

	if err := conf.Marshal(c); err != nil {
		return nil, err
	}

	sm := conf.ToStringMap()

	return yaml.Marshal(sm)
}

// Endpoint returns a listener endpoint for OTEL components.
func Endpoint(port int) string {
	return fmt.Sprintf("${env:MY_POD_IP}:%d", port)
}

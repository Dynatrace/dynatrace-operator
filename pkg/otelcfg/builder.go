// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package otelcfg

import (
	"maps"
	"slices"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/service/pipelines"
)

var allSignals = []pipeline.Signal{pipeline.SignalMetrics, pipeline.SignalTraces, pipeline.SignalLogs}

type Builder struct {
	cfg       *Config
	pipelines pipelines.Config
}

func NewBuilder() *Builder {
	return &Builder{
		cfg: &Config{
			Receivers:  map[component.ID]component.Config{},
			Processors: map[component.ID]component.Config{},
			Exporters:  map[component.ID]component.Config{},
			Extensions: map[component.ID]component.Config{},
			Service:    ServiceConfig{Pipelines: pipelines.Config{}},
		},
		pipelines: pipelines.Config{},
	}
}

// Receiver registers the receiver with the signal pipelines, in registration order.
// The signals can be used to limit the registration to the respective pipelines.
func (b *Builder) Receiver(id component.ID, cfg component.Config, signals ...pipeline.Signal) *Builder {
	b.cfg.Receivers[id] = cfg
	b.forEachSignal(signals, func(cfg *pipelines.PipelineConfig) {
		cfg.Receivers = append(cfg.Receivers, id)
	})

	return b
}

// Processor registers the processor with the signal pipelines, in registration order.
// The signals can be used to limit the registration to the respective pipelines.
func (b *Builder) Processor(id component.ID, cfg component.Config, signals ...pipeline.Signal) *Builder {
	b.cfg.Processors[id] = cfg
	b.forEachSignal(signals, func(cfg *pipelines.PipelineConfig) {
		cfg.Processors = append(cfg.Processors, id)
	})

	return b
}

// Exporter registers the exporter with all signal pipelines, in registration order.
func (b *Builder) Exporter(id component.ID, cfg component.Config) *Builder {
	b.cfg.Exporters[id] = cfg
	b.forEachSignal(allSignals, func(cfg *pipelines.PipelineConfig) {
		cfg.Exporters = append(cfg.Exporters, id)
	})

	return b
}

// Extension registers the extension and enables it in the service, in registration order.
func (b *Builder) Extension(id component.ID, cfg component.Config) *Builder {
	b.cfg.Extensions[id] = cfg
	b.cfg.Service.Extensions = append(b.cfg.Service.Extensions, id)

	return b
}

// Build generates a Config for the given signals from the working set of the builder.
func (b *Builder) Build(signals ...pipeline.Signal) *Config {
	buildCfg := &Config{
		Receivers:  maps.Clone(b.cfg.Receivers),
		Exporters:  maps.Clone(b.cfg.Exporters),
		Processors: maps.Clone(b.cfg.Processors),
		Extensions: maps.Clone(b.cfg.Extensions),
		Service: ServiceConfig{
			Extensions: slices.Clone(b.cfg.Service.Extensions),
			Pipelines:  pipelines.Config{},
		},
	}

	if len(signals) == 0 {
		signals = allSignals
	}

	for _, signal := range signals {
		key := pipeline.NewID(signal)
		cfg := b.pipelines[key]

		if cfg != nil && len(cfg.Receivers) > 0 && len(cfg.Processors) > 0 {
			buildCfg.Service.Pipelines[key] = new(*cfg)
		}
	}

	return buildCfg
}

func (b *Builder) forEachSignal(signals []pipeline.Signal, f func(cfg *pipelines.PipelineConfig)) {
	if len(signals) == 0 {
		signals = allSignals
	}

	for _, signal := range signals {
		key := pipeline.NewID(signal)

		cfg := b.pipelines[key]
		if cfg == nil {
			cfg = &pipelines.PipelineConfig{}
		}

		f(cfg)

		b.pipelines[key] = cfg
	}
}

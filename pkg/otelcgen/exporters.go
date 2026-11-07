// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package otelcgen

import (
	"go.opentelemetry.io/collector/component"
)

var OTLPHTTPID = component.MustNewID("otlp_http")

type ExporterOpts struct {
	CAFile                   string
	APIToken                 string
	IncludeSystemCACertsPool bool
	BearerTokenAuth          bool
}

func (b *Builder) OTLPHTTPExporter(endpoint string, opts ExporterOpts) *Builder {
	// https://github.com/open-telemetry/opentelemetry-collector/tree/v0.157.0/exporter/otlphttpexporter
	cfg := map[string]any{
		"endpoint": endpoint,
	}

	if opts.APIToken != "" {
		cfg["headers"] = map[string]string{"Authorization": "Api-Token " + opts.APIToken}
	}

	if opts.CAFile != "" {
		// https://github.com/open-telemetry/opentelemetry-collector/blob/v0.157.0/config/configtls/configtls.go#L34
		cfg["tls"] = map[string]any{
			"ca_file":                      opts.CAFile,
			"include_system_ca_certs_pool": opts.IncludeSystemCACertsPool,
		}
	}

	if opts.BearerTokenAuth {
		// https://github.com/open-telemetry/opentelemetry-collector/blob/v0.157.0/config/configauth/configauth.go#L25
		cfg["auth"] = map[string]any{
			"authenticator": BearerAuthID.String(),
		}
		// https://github.com/open-telemetry/opentelemetry-collector/blob/v0.157.0/exporter/exporterhelper/internal/queuebatch/config.go#L19
		cfg["sending_queue"] = map[string]any{
			// https://github.com/open-telemetry/opentelemetry-collector/blob/v0.157.0/exporter/exporterhelper/internal/queuebatch/config.go#L92
			"batch": map[string]any{
				"flush_timeout": "10s",
				"max_size":      5000,
				"min_size":      500,
			},
		}
	}

	return b.Exporter(OTLPHTTPID, cfg)
}

// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package otelcgen

import "go.opentelemetry.io/collector/component"

var (
	healthCheck = component.MustNewID("health_check")
)

func (c *Config) buildExtensions() map[component.ID]component.Config {
	return map[component.ID]component.Config{
		healthCheck: &ServerConfig{
			Endpoint: c.buildEndpoint(ExtensionsHealthCheckPort),
		},
	}
}

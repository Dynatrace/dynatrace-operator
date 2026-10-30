// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package otelcgen

import (
	"go.opentelemetry.io/collector/component"
)

const ExtensionsHealthCheckPort = 13133

var (
	HealthCheckID = component.MustNewID("health_check")
	BearerAuthID  = component.MustNewID("bearertokenauth")
)

// HealthCheck builds the config of the health_check extension, which listens on the IP of the pod.
func (b *Builder) HealthCheck() *Builder {
	// https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.157.0/extension/healthcheckextension
	return b.Extension(HealthCheckID, map[string]any{
		"endpoint": Endpoint(ExtensionsHealthCheckPort),
	})
}

func (b *Builder) BearerTokenAuth(filename string) *Builder {
	// https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.157.0/extension/bearertokenauthextension
	return b.Extension(BearerAuthID, map[string]any{
		"scheme":   "Api-Token",
		"filename": filename,
	})
}

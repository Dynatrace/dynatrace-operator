// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package gateway

import (
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/otelcgen"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pipeline"
)

// Component IDs are typed constants so that MustNewID validates the names at program start,
// and the string representations derive from a single source.
// The IDs of the components shared with the other collectors come from otelcgen.
var (
	otlpID            = component.MustNewID("otlp")
	metricStartTimeID = component.MustNewID("metric_start_time")
)

// prometheusLabelRenames maps the semantic convention name of an attribute to the name Prometheus
// exposes it as, because the characters of a Prometheus label name are restricted.
var prometheusLabelRenames = []struct{ semconv, prometheus string }{
	{"http.request.method", "http_request_method"},
	{"http.response.status_code", "http_response_status_code"},
	{"network.protocol.name", "network_protocol_name"},
	{"network.protocol.version", "network_protocol_version"},
	{"rpc.method", "rpc_method"},
	{"rpc.response.status_code", "rpc_response_status_code"},
	{"rpc.system.name", "rpc_system_name"},
	{"server.address", "server_address"},
	{"server.port", "server_port"},
	{"url.scheme", "url_scheme"},
}

func buildConfig(data gatewayConfigData) *otelcgen.Config {
	return otelcgen.NewBuilder().
		Receiver(otlpID, map[string]any{
			"protocols": map[string]any{
				"grpc": map[string]any{
					"endpoint": otelcgen.Endpoint(otelcgen.OTLPReceiverGRPCPort),
				},
			},
		}).
		MemoryLimiter(otelcgen.PrometheusMonitoringMemoryLimiter).
		Processor(metricStartTimeID, otelcgen.Empty).
		Processor(otelcgen.CumulativeToDeltaID, map[string]any{
			"initial_value": "drop",
			"max_staleness": "10m",
		}).
		// The annotations are extracted before the static resource attributes and the facts after them, so that
		// the precedence is: annotations > static resource attributes > facts. See otelcgen.K8sAttributesAnnotationsID.
		Processor(otelcgen.K8sAttributesAnnotationsID, otelcgen.K8sAttributes(otelcgen.K8sAttributesOpts{
			Metadata:    []string{},
			Annotations: true,
		})).
		StaticResourceAttributes(data.ResourceAttributes).
		Processor(otelcgen.K8sAttributesFactsID, otelcgen.K8sAttributes(otelcgen.K8sAttributesOpts{
			// k8s.container.name is only extracted by the gateway.
			Metadata: append(otelcgen.K8sFactsMetadata(true), "k8s.container.name"),
		})).
		Processor(otelcgen.TransformID, buildTransformConfig()).
		OTLPHTTPExporter(data.Endpoint, otelcgen.ExporterOpts{
			CAFile:          data.CustomCAPath,
			BearerTokenAuth: true,
		}).
		HealthCheck().
		BearerTokenAuth(tokenMountPath + "/" + tokenFileName).
		Build(pipeline.SignalMetrics)
}

// buildTransformConfig assembles the transform processor config. The metadata.dynatrace.com merge comes first
// and overwrites existing attributes, everything after it only fills gaps, so the annotations always take precedence.
func buildTransformConfig() component.Config {
	return otelcgen.Transform(
		[]pipeline.Signal{pipeline.SignalMetrics},
		otelcgen.ResourceStatements(otelcgen.MergeDynatraceAnnotations()),
		map[string]any{
			"context":    "datapoint",
			"statements": prometheusLabelsToSemconv(),
		},
		otelcgen.ResourceStatements(otelcgen.WorkloadFromOwners(true), otelcgen.DeleteOwnerFacts(true)),
		otelcgen.ResourceStatements(scrubOTelScope()),
		otelcgen.ResourceStatements([]string{
			`set(attributes["k8s.cluster.name"], "${env:K8S_CLUSTER_NAME}") where attributes["k8s.cluster.name"] == nil and Len("${env:K8S_CLUSTER_NAME}") > 0`,
		}),
	)
}

// prometheusLabelsToSemconv renames the attributes that Prometheus exposes with underscores to the names of the
// semantic conventions. These only exist on scraped metrics.
func prometheusLabelsToSemconv() []string {
	statements := make([]string, 0, len(prometheusLabelRenames))

	for _, rename := range prometheusLabelRenames {
		statements = append(
			statements,
			fmt.Sprintf(`set(attributes[%q], attributes[%q]) where attributes[%q] != nil`, rename.semconv, rename.prometheus, rename.prometheus),
			fmt.Sprintf(`delete_key(attributes, %q)`, rename.prometheus),
		)
	}

	return statements
}

// scrubOTelScope deletes the attributes that describe how the metrics were scraped, which are of no use downstream.
// They must not be deleted for telemetry that is sent to the collector directly, as it can legitimately carry them.
func scrubOTelScope() []string {
	keys := []string{"processor", "otel.signal", "otel.scope.name", "otel.scope.version"}
	statements := make([]string, 0, len(keys))

	for _, key := range keys {
		statements = append(statements, fmt.Sprintf(`delete_key(attributes, %q)`, key))
	}

	return statements
}

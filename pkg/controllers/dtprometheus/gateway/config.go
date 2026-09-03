// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package gateway

import (
	"fmt"
	"maps"
	"slices"
	"strconv"

	"github.com/Dynatrace/dynatrace-operator/pkg/otelcgen"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/service/extensions"
	"go.opentelemetry.io/collector/service/pipelines"
)

// Component and pipeline IDs are typed constants so that MustNewID/MustNewIDWithName
// validate the names at program start, and the string representations derive from a single source.
var (
	otlpID              = component.MustNewID("otlp")
	memoryLimiterID     = component.MustNewID("memory_limiter")
	metricStartTimeID   = component.MustNewID("metric_start_time")
	cumulativeToDeltaID = component.MustNewID("cumulative_to_delta")
	k8sAttributesID     = component.MustNewID("k8s_attributes")
	transformID         = component.MustNewID("transform")
	otlpHTTPID          = component.MustNewID("otlp_http")
	healthCheckID       = component.MustNewID("health_check")
	metricsSignalID     = pipeline.NewID(pipeline.SignalMetrics)
)

// buildGatewayOTelConfig assembles the OTel Collector relay config as an otelcgen.Config.
// The caller marshals it to YAML via cfg.Marshal().
func buildGatewayOTelConfig(data gatewayConfigData) *otelcgen.Config {
	processors, processorIDs := buildProcessorMap(data)

	return &otelcgen.Config{
		Receivers: map[component.ID]component.Config{
			otlpID: map[string]any{
				"protocols": map[string]any{
					"grpc": map[string]any{
						"endpoint": "${env:MY_POD_IP}:4317",
					},
				},
			},
		},
		Processors: processors,
		Exporters: map[component.ID]component.Config{
			otlpHTTPID: buildOTLPHTTPExporter(data),
		},
		Extensions: map[component.ID]component.Config{
			healthCheckID: map[string]any{
				"endpoint": "${env:MY_POD_IP}:13133",
			},
		},
		Service: otelcgen.ServiceConfig{
			Extensions: extensions.Config{healthCheckID},
			Pipelines: pipelines.Config{
				metricsSignalID: &pipelines.PipelineConfig{
					Receivers:  []component.ID{otlpID},
					Processors: processorIDs,
					Exporters:  []component.ID{otlpHTTPID},
				},
			},
		},
	}
}

func buildProcessorMap(data gatewayConfigData) (map[component.ID]component.Config, []component.ID) {
	m := map[component.ID]component.Config{
		memoryLimiterID: map[string]any{
			"check_interval":         "1s",
			"limit_percentage":       95,
			"spike_limit_percentage": 5,
		},
		metricStartTimeID: map[string]any{},
		cumulativeToDeltaID: map[string]any{
			"initial_value": "drop",
			"max_staleness": "10m",
		},
		k8sAttributesID: buildK8sAttributesConfig(),
		transformID:     buildTransformConfig(data.ResourceAttributes),
	}

	ids := []component.ID{
		memoryLimiterID,
		metricStartTimeID,
		cumulativeToDeltaID,
		k8sAttributesID,
		transformID,
	}

	return m, ids
}

func buildOTLPHTTPExporter(data gatewayConfigData) component.Config {
	exp := map[string]any{
		"endpoint": data.Endpoint,
		"headers": map[string]any{
			"Authorization": "Api-Token ${env:DT_API_TOKEN}",
		},
		"sending_queue": map[string]any{
			"batch": map[string]any{
				"flush_timeout": "10s",
				"max_size":      5000,
				"min_size":      500,
			},
		},
	}

	if data.CustomCAPath != "" {
		exp["tls"] = map[string]any{"ca_file": data.CustomCAPath}
	}

	return exp
}

func buildK8sAttributesConfig() component.Config {
	return map[string]any{
		"extract": map[string]any{
			"annotations": []map[string]any{
				{
					"from":      "pod",
					"key_regex": `metadata.dynatrace.com/(.*)`,
					"tag_name":  "$$1",
				},
				{
					"from":     "pod",
					"key":      "metadata.dynatrace.com",
					"tag_name": "metadata.dynatrace.com",
				},
			},
			"metadata": []string{
				"k8s.pod.name",
				"k8s.pod.uid",
				"k8s.pod.ip",
				"k8s.deployment.name",
				"k8s.replicaset.name",
				"k8s.statefulset.name",
				"k8s.daemonset.name",
				"k8s.job.name",
				"k8s.cronjob.name",
				"k8s.namespace.name",
				"k8s.node.name",
				"k8s.cluster.uid",
				"k8s.container.name",
				"k8s.deployment.uid",
				"k8s.replicaset.uid",
				"k8s.statefulset.uid",
				"k8s.daemonset.uid",
				"k8s.job.uid",
				"k8s.cronjob.uid",
			},
		},
		"pod_association": []map[string]any{
			{
				"sources": []map[string]any{
					{"from": "resource_attribute", "name": "k8s.pod.name"},
					{"from": "resource_attribute", "name": "k8s.namespace.name"},
				},
			},
			{
				"sources": []map[string]any{
					{"from": "resource_attribute", "name": "k8s.pod.ip"},
				},
			},
			{
				"sources": []map[string]any{
					{"from": "resource_attribute", "name": "k8s.pod.uid"},
				},
			},
			{
				"sources": []map[string]any{
					{"from": "connection"},
				},
			},
		},
	}
}

// buildTransformConfig assembles the transform processor config.
func buildTransformConfig(resourceAttributes map[string]string) component.Config {
	blocks := []map[string]any{
		{
			"context": "datapoint",
			"statements": []string{
				`set(attributes["http.request.method"], attributes["http_request_method"]) where attributes["http_request_method"] != nil`,
				`delete_key(attributes, "http_request_method")`,
				`set(attributes["http.response.status_code"], attributes["http_response_status_code"]) where attributes["http_response_status_code"] != nil`,
				`delete_key(attributes, "http_response_status_code")`,
				`set(attributes["network.protocol.name"], attributes["network_protocol_name"]) where attributes["network_protocol_name"] != nil`,
				`delete_key(attributes, "network_protocol_name")`,
				`set(attributes["network.protocol.version"], attributes["network_protocol_version"]) where attributes["network_protocol_version"] != nil`,
				`delete_key(attributes, "network_protocol_version")`,
				`set(attributes["rpc.method"], attributes["rpc_method"]) where attributes["rpc_method"] != nil`,
				`delete_key(attributes, "rpc_method")`,
				`set(attributes["rpc.response.status_code"], attributes["rpc_response_status_code"]) where attributes["rpc_response_status_code"] != nil`,
				`delete_key(attributes, "rpc_response_status_code")`,
				`set(attributes["rpc.system.name"], attributes["rpc_system_name"]) where attributes["rpc_system_name"] != nil`,
				`delete_key(attributes, "rpc_system_name")`,
				`set(attributes["server.address"], attributes["server_address"]) where attributes["server_address"] != nil`,
				`delete_key(attributes, "server_address")`,
				`set(attributes["server.port"], attributes["server_port"]) where attributes["server_port"] != nil`,
				`delete_key(attributes, "server_port")`,
				`set(attributes["url.scheme"], attributes["url_scheme"]) where attributes["url_scheme"] != nil`,
				`delete_key(attributes, "url_scheme")`,
			},
		},
		{
			"context": "resource",
			"statements": []string{
				`set(attributes["k8s.workload.name"], attributes["k8s.replicaset.name"]) where IsString(attributes["k8s.replicaset.name"])`,
				`set(attributes["k8s.workload.name"], attributes["k8s.statefulset.name"]) where IsString(attributes["k8s.statefulset.name"])`,
				`set(attributes["k8s.workload.name"], attributes["k8s.job.name"]) where IsString(attributes["k8s.job.name"])`,
				`set(attributes["k8s.workload.name"], attributes["k8s.deployment.name"]) where IsString(attributes["k8s.deployment.name"])`,
				`set(attributes["k8s.workload.name"], attributes["k8s.daemonset.name"]) where IsString(attributes["k8s.daemonset.name"])`,
				`set(attributes["k8s.workload.name"], attributes["k8s.cronjob.name"]) where IsString(attributes["k8s.cronjob.name"])`,
				`set(attributes["k8s.workload.kind"], "replicaset") where IsString(attributes["k8s.replicaset.name"])`,
				`set(attributes["k8s.workload.kind"], "statefulset") where IsString(attributes["k8s.statefulset.name"])`,
				`set(attributes["k8s.workload.kind"], "job") where IsString(attributes["k8s.job.name"])`,
				`set(attributes["k8s.workload.kind"], "deployment") where IsString(attributes["k8s.deployment.name"])`,
				`set(attributes["k8s.workload.kind"], "daemonset") where IsString(attributes["k8s.daemonset.name"])`,
				`set(attributes["k8s.workload.kind"], "cronjob") where IsString(attributes["k8s.cronjob.name"])`,
				`set(attributes["k8s.workload.uid"], attributes["k8s.replicaset.uid"]) where IsString(attributes["k8s.replicaset.uid"])`,
				`set(attributes["k8s.workload.uid"], attributes["k8s.statefulset.uid"]) where IsString(attributes["k8s.statefulset.uid"])`,
				`set(attributes["k8s.workload.uid"], attributes["k8s.job.uid"]) where IsString(attributes["k8s.job.uid"])`,
				`set(attributes["k8s.workload.uid"], attributes["k8s.deployment.uid"]) where IsString(attributes["k8s.deployment.uid"])`,
				`set(attributes["k8s.workload.uid"], attributes["k8s.daemonset.uid"]) where IsString(attributes["k8s.daemonset.uid"])`,
				`set(attributes["k8s.workload.uid"], attributes["k8s.cronjob.uid"]) where IsString(attributes["k8s.cronjob.uid"])`,
				`delete_key(attributes, "k8s.statefulset.name")`,
				`delete_key(attributes, "k8s.replicaset.name")`,
				`delete_key(attributes, "k8s.job.name")`,
				`delete_key(attributes, "k8s.deployment.name")`,
				`delete_key(attributes, "k8s.daemonset.name")`,
				`delete_key(attributes, "k8s.cronjob.name")`,
				`delete_key(attributes, "k8s.statefulset.uid")`,
				`delete_key(attributes, "k8s.replicaset.uid")`,
				`delete_key(attributes, "k8s.deployment.uid")`,
				`delete_key(attributes, "k8s.daemonset.uid")`,
				`delete_key(attributes, "k8s.job.uid")`,
				`delete_key(attributes, "k8s.cronjob.uid")`,
			},
		},
		{
			"context": "resource",
			"statements": []string{
				`delete_key(attributes, "processor")`,
				`delete_key(attributes, "otel.signal")`,
				`delete_key(attributes, "otel.scope.name")`,
				`delete_key(attributes, "otel.scope.version")`,
			},
		},
		{
			"context": "resource",
			"statements": []string{
				`set(attributes["k8s.cluster.name"], "${env:K8S_CLUSTER_NAME}") where attributes["k8s.cluster.name"] == nil and Len("${env:K8S_CLUSTER_NAME}") > 0`,
			},
		},
	}

	// resourceAttributes are only a fallback: the "== nil" guard in each set() statement
	// and the annotation merge below (which runs after and always overwrites) make sure
	// pod/workload/namespace annotations always take precedence.
	if len(resourceAttributes) > 0 {
		blocks = append(blocks, map[string]any{
			"context":    "resource",
			"statements": buildResourceAttributeStatements(resourceAttributes),
		})
	}

	blocks = append(blocks, map[string]any{
		"context": "resource",
		"statements": []string{
			`merge_maps(attributes, ParseJSON(attributes["metadata.dynatrace.com"]), "upsert") where IsMatch(attributes["metadata.dynatrace.com"], "^\\{")`,
			`delete_key(attributes, "metadata.dynatrace.com")`,
		},
	})

	return map[string]any{"metric_statements": blocks}
}

// buildResourceAttributeStatements returns sorted set() statements for resourceAttributes.
func buildResourceAttributeStatements(attrs map[string]string) []string {
	keys := slices.Sorted(maps.Keys(attrs))
	statements := make([]string, 0, len(keys))

	for _, k := range keys {
		quotedKey := strconv.Quote(k)
		quotedVal := strconv.Quote(attrs[k])
		statements = append(statements, fmt.Sprintf(
			`set(attributes[%s], %s) where attributes[%s] == nil`,
			quotedKey, quotedVal, quotedKey,
		))
	}

	return statements
}

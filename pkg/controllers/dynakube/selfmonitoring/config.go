// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package selfmonitoring

import (
	"maps"
	"slices"

	"github.com/Dynatrace/dynatrace-operator/pkg/otelcgen"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/service/extensions"
	"go.opentelemetry.io/collector/service/pipelines"
)

// Component and pipeline IDs are typed constants so that MustNewID validates the names
// at program start, and the string representations derive from a single source.
var (
	otlpID          = component.MustNewID("otlp")
	memoryLimiterID = component.MustNewID("memory_limiter")
	renameID        = component.MustNewIDWithName("transform", "selfmon_rename")
	k8sAttributesID = component.MustNewID("k8s_attributes")
	transformID     = component.MustNewID("transform")
	resourceAttrsID = component.MustNewIDWithName("resource", "dynakube")
	otlpHTTPID      = component.MustNewID("otlp_http")
	healthCheckID   = component.MustNewID("health_check")
	bearerAuthID    = component.MustNewID("bearertokenauth")
	metricsSignalID = pipeline.NewID(pipeline.SignalMetrics)
)

// selfMonitoringConfigData holds the inputs to the collector config that are derived from the DynaKube.
type selfMonitoringConfigData struct {
	// Endpoint is the tenant OTLP endpoint. The tenant is always used instead of an ActiveGate
	// so that self-monitoring data is not lost when the local ActiveGate is in trouble.
	Endpoint string

	// Namespace restricts the k8s_attributes processor to the DynaKube namespace. The
	// self-monitoring service account can only watch pods there.
	Namespace string

	// CustomCAPath is the path to the trusted CA bundle for the exporter, empty if none is configured.
	CustomCAPath string

	// ResourceAttributes are the DynaKube's .spec.resourceAttributes, added to all self-monitoring data.
	ResourceAttributes map[string]string
}

// buildSelfMonitoringOTelConfig assembles the collector config as an otelcgen.Config.
// The caller marshals it to YAML via cfg.Marshal().
//
// There is no cumulative_to_delta processor on purpose: senders push delta temporality, which
// keeps the collector stateless so any replica can receive data from any sender.
func buildSelfMonitoringOTelConfig(data selfMonitoringConfigData) *otelcgen.Config {
	cfg := &otelcgen.Config{
		Receivers: map[component.ID]component.Config{
			otlpID: map[string]any{
				"protocols": map[string]any{
					"http": map[string]any{
						"endpoint": "${env:MY_POD_IP}:4318",
					},
				},
			},
		},
		Processors: map[component.ID]component.Config{
			memoryLimiterID: map[string]any{
				"check_interval":         "1s",
				"limit_percentage":       80,
				"spike_limit_percentage": 20,
			},
			renameID: map[string]any{
				"metric_statements": []map[string]any{
					{"statements": []string{`replace_all_patterns(resource.attributes, "key", "_", ".")`}},
				},
			},
			k8sAttributesID: buildK8sAttributesProcessor(data),
			transformID:     buildTransformProcessor(),
		},
		Exporters: map[component.ID]component.Config{
			otlpHTTPID: buildOTLPHTTPExporter(data),
		},
		Extensions: map[component.ID]component.Config{
			healthCheckID: map[string]any{
				"endpoint": "${env:MY_POD_IP}:13133",
			},
			bearerAuthID: map[string]any{
				"filename": tokenMountPath + "/" + tokenFileName,
				"scheme":   "Api-Token",
			},
		},
		Service: otelcgen.ServiceConfig{
			Extensions: extensions.Config{healthCheckID, bearerAuthID},
			Pipelines: pipelines.Config{
				metricsSignalID: &pipelines.PipelineConfig{
					Receivers: []component.ID{otlpID},
					Processors: []component.ID{
						memoryLimiterID,
						renameID,
						k8sAttributesID,
						transformID,
					},
					Exporters: []component.ID{otlpHTTPID},
				},
			},
		},
	}

	if len(data.ResourceAttributes) > 0 {
		cfg.Processors[resourceAttrsID] = buildResourceAttributesProcessor(data.ResourceAttributes)
		metrics := cfg.Service.Pipelines[metricsSignalID]
		metrics.Processors = append(metrics.Processors, resourceAttrsID)
	}

	return cfg
}

// buildResourceAttributesProcessor adds the DynaKube's resource attributes with action "insert" as
// the last processor, so they only fill gaps and never overwrite attributes set by the sender,
// the k8s_attributes processor or the metadata annotations.
func buildResourceAttributesProcessor(attrs map[string]string) component.Config {
	attributes := make([]map[string]any, 0, len(attrs))

	// Sorted for a stable config, otherwise the config hash would change and roll the pods.
	for _, key := range slices.Sorted(maps.Keys(attrs)) {
		attributes = append(attributes, map[string]any{
			"key":    key,
			"value":  attrs[key],
			"action": "insert",
		})
	}

	return map[string]any{"attributes": attributes}
}

func buildOTLPHTTPExporter(data selfMonitoringConfigData) component.Config {
	exp := map[string]any{
		"endpoint": data.Endpoint,
		"auth": map[string]any{
			"authenticator": bearerAuthID.String(),
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

func buildK8sAttributesProcessor(data selfMonitoringConfigData) component.Config {
	return map[string]any{
		"filter": map[string]any{
			"namespace": data.Namespace,
		},
		"extract": map[string]any{
			"annotations": []map[string]any{
				{
					"from":      "pod",
					"key_regex": "metadata.dynatrace.com/(.*)",
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
		"passthrough": false,
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

func buildTransformProcessor() component.Config {
	return map[string]any{
		"metric_statements": []map[string]any{
			{
				"context": "datapoint",
				"statements": []string{
					`set(resource.attributes["k8s.pod.name"], attributes["k8s_pod_name"]) where attributes["k8s_pod_name"] != nil`,
					`set(resource.attributes["k8s.namespace.name"], attributes["k8s_namespace_name"]) where attributes["k8s_namespace_name"] != nil`,
					`set(resource.attributes["k8s.pod.ip"], attributes["k8s_pod_ip"]) where attributes["k8s_pod_ip"] != nil`,
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
			{
				"context": "resource",
				"statements": []string{
					`merge_maps(attributes, ParseJSON(attributes["metadata.dynatrace.com"]), "upsert") where IsMatch(attributes["metadata.dynatrace.com"], "^\\{")`,
					`delete_key(attributes, "metadata.dynatrace.com")`,
				},
			},
		},
	}
}

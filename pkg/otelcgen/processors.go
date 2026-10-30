// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package otelcgen

import (
	"slices"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pipeline"
)

// https://github.com/open-telemetry/opentelemetry-collector/blob/v0.157.0/processor/batchprocessor/config.go#L16
type batchProcessorConfig struct {
	// Timeout sets the time after which a batch will be sent regardless of size.
	// When this is set to zero, batched data will be sent immediately.
	Timeout time.Duration `mapstructure:"timeout"`

	// SendBatchSize is the size of a batch which after hit, will trigger it to be sent.
	// When this is set to zero, the batch size is ignored and data will be sent immediately
	// subject to only send_batch_max_size.
	SendBatchSize uint32 `mapstructure:"send_batch_size"`

	// SendBatchMaxSize is the maximum size of a batch. It must be larger than SendBatchSize.
	// Larger batches are split into smaller units.
	// Default value is 0, that means no maximum size.
	SendBatchMaxSize uint32 `mapstructure:"send_batch_max_size"`
}

// https://github.com/open-telemetry/opentelemetry-collector/blob/v0.157.0/internal/memorylimiter/config.go#L29
type MemoryLimiterOpts struct {
	// MemoryLimitPercentage is the maximum amount of memory, in %, targeted to be
	// allocated by the process. The fixed memory settings MemoryLimitMiB has a higher precedence.
	LimitPercentage uint32
	// MemorySpikePercentage is the maximum, in percents against the total memory,
	// spike expected between the measurements of memory usage.
	SpikePercentage uint32
}

// PrometheusMonitoringMemoryLimiter are the memory_limiter settings of the scraper and gateway of PrometheusMonitoring.
var PrometheusMonitoringMemoryLimiter = MemoryLimiterOpts{LimitPercentage: 95, SpikePercentage: 5}

// MemoryLimiter builds the config of the memory_limiter processor.
func MemoryLimiter(opts MemoryLimiterOpts) component.Config {
	return map[string]any{
		"check_interval":         "1s",
		"limit_percentage":       opts.LimitPercentage,
		"spike_limit_percentage": opts.SpikePercentage,
	}
}

// More details, about how to configure `processors,` can be found
// https://github.com/open-telemetry/opentelemetry-collector/blob/main/processor/batchprocessor/README.md
var (
	// K8sAttributesAnnotationsID and K8sAttributesFactsID must stay two separate processor instances:
	// k8sattributesprocessor's setResourceAttribute helper is insert-only (never overwrites an
	// already-set, non-empty attribute) for both its metadata and annotation extraction rules,
	// regardless of pipeline position. That's the mechanism that lets resource/staticAttrs sit
	// between them: annotations claim their keys first (so pod values always win over the static
	// DynaKube default), then staticAttrs fills gaps and claims keys before the facts extraction
	// runs (so the static default always wins over built-in facts). See buildPipelineProcessors.
	K8sAttributesAnnotationsID = component.MustNewIDWithName("k8s_attributes", "annotations")
	K8sAttributesFactsID       = component.MustNewIDWithName("k8s_attributes", "facts")
	TransformID                = component.MustNewID("transform")
	transformPodIP             = component.MustNewIDWithName("transform", "add-pod-ip")
	StaticResourceAttrsID      = component.MustNewIDWithName("resource", "staticAttrs")
	batch                      = component.MustNewType("batch")
	batchTraces                = component.NewIDWithName(batch, "traces")
	batchMetrics               = component.NewIDWithName(batch, "metrics")
	batchLogs                  = component.NewIDWithName(batch, "logs")
	MemoryLimiterID            = component.MustNewID("memory_limiter")
	CumulativeToDeltaID        = component.MustNewID("cumulative_to_delta")
)

// PodAssociation returns the pod_association config of the k8sattributes processor, so each
// instance can independently identify which pod a resource belongs to.
func PodAssociation() []map[string]any {
	return []map[string]any{
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
	}
}

// StaticResourceAttributes builds the resource/staticAttrs processor config, using action "insert" so it never
// overwrites a value already claimed by K8sAttributesAnnotationsID (per-pod annotations must win), while still being
// able to claim a key before K8sAttributesFactsID runs (so the static DynaKube default wins over the built-in fact).
// See buildPipelineProcessors.
func StaticResourceAttributes(resourceAttributes map[string]string) map[string]any {
	keys := make([]string, 0, len(resourceAttributes))
	for k := range resourceAttributes {
		keys = append(keys, k)
	}

	slices.Sort(keys)

	attributes := make([]map[string]any, 0, len(keys))
	for _, k := range keys {
		attributes = append(attributes, map[string]any{
			"key":    k,
			"value":  resourceAttributes[k],
			"action": "insert",
		})
	}

	return map[string]any{
		"attributes": attributes,
	}
}

func buildTransform() map[string]any {
	return Transform(
		[]pipeline.Signal{pipeline.SignalTraces, pipeline.SignalMetrics, pipeline.SignalLogs},
		dynatraceTransformations()...,
	)
}

func buildTransformPodIP() map[string]any {
	return map[string]any{
		"error_mode": "ignore",
		"trace_statements": []map[string]any{
			{
				"context":    "resource",
				"statements": []string{"set(attributes[\"k8s.pod.ip\"], attributes[\"ip\"]) where attributes[\"k8s.pod.ip\"] == nil"},
			},
		},
	}
}

// dynatraceTransformations enriches the resource attributes of all signals: the workload is derived from the
// facts about the owners of the pod, which are deleted afterward, and the cluster attributes are set if enabled.
func dynatraceTransformations() []map[string]any {
	groups := [][]string{
		MergeDynatraceAnnotations(),
		WorkloadFromOwners(false),
		// TODO: decide whether we should keep these here
		clusterAttributeStatements(),
		DeleteOwnerFacts(false),
	}

	return []map[string]any{ResourceStatements(groups...)}
}

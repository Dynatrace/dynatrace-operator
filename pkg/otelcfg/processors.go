// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package otelcfg

import (
	"slices"

	"go.opentelemetry.io/collector/component"
)

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

func (b *Builder) MemoryLimiter(opts MemoryLimiterOpts) *Builder {
	return b.Processor(MemoryLimiterID, MemoryLimiter(opts))
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
	StaticResourceAttrsID      = component.MustNewIDWithName("resource", "staticAttrs")
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
func staticResourceAttributes(resourceAttributes map[string]string) map[string]any {
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

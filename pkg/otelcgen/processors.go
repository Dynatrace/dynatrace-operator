// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package otelcgen

import (
	"slices"

	"go.opentelemetry.io/collector/component"
)

// BatchConfig represents common attributes to config batch processor:
// inspired by
// https://github.com/open-telemetry/opentelemetry-collector/blob/main/processor/batchprocessor/config.go#L16
type BatchConfig struct {
	Timeout          string `mapstructure:"timeout"`
	SendBatchSize    uint32 `mapstructure:"send_batch_size"`
	SendBatchMaxSize uint32 `mapstructure:"send_batch_max_size"`
}

// MemoryLimiter represents common attributes for memory limiter
// inspired by
// https://github.com/open-telemetry/opentelemetry-collector/blob/internal/memorylimiter/v0.117.0/internal/memorylimiter/config.go#L23
type MemoryLimiter struct {
	CheckInterval         string `mapstructure:"check_interval"`
	MemoryLimitPercentage uint32 `mapstructure:"limit_percentage"`
	MemorySpikePercentage uint32 `mapstructure:"spike_limit_percentage"`
}

// More details, about how to configure `processors,` can be found
// https://github.com/open-telemetry/opentelemetry-collector/blob/main/processor/batchprocessor/README.md
var (
	// k8sattributesAnnotations and k8sattributesFacts must stay two separate processor instances:
	// k8sattributesprocessor's setResourceAttribute helper is insert-only (never overwrites an
	// already-set, non-empty attribute) for both its metadata and annotation extraction rules,
	// regardless of pipeline position. That's the mechanism that lets resource/staticAttrs sit
	// between them: annotations claim their keys first (so pod values always win over the static
	// DynaKube default), then staticAttrs fills gaps and claims keys before the facts extraction
	// runs (so the static default always wins over built-in facts). See buildPipelineProcessors.
	k8sattributesAnnotations = component.MustNewIDWithName("k8sattributes", "annotations")
	k8sattributesFacts       = component.MustNewIDWithName("k8sattributes", "facts")
	transform                = component.MustNewID("transform")
	transformPodIP           = component.MustNewIDWithName("transform", "add-pod-ip")
	staticResourceAttrs      = component.MustNewIDWithName("resource", "staticAttrs")
	batch                    = component.MustNewType("batch")
	batchTraces              = component.NewIDWithName(batch, "traces")
	batchMetrics             = component.NewIDWithName(batch, "metrics")
	batchLogs                = component.NewIDWithName(batch, "logs")
	memoryLimiter            = component.MustNewID("memory_limiter")
	cumulativeToDelta        = component.MustNewID("cumulativetodelta")

	defaultK8Sattributes = []string{
		"k8s.cluster.uid",
		"k8s.node.name",
		"k8s.namespace.name",
		"k8s.pod.name",
		"k8s.pod.uid",
		"k8s.pod.ip",
		"k8s.deployment.name",
		"k8s.replicaset.name",
		"k8s.statefulset.name",
		"k8s.daemonset.name",
		"k8s.cronjob.name",
		"k8s.job.name",
	}
)

func (c *Config) buildProcessors() map[component.ID]component.Config {
	processors := map[component.ID]component.Config{
		cumulativeToDelta: map[string]any{},
		k8sattributesAnnotations: map[string]any{
			"extract": map[string]any{
				// k8sattributesprocessor treats an omitted "metadata" field as "use its
				// built-in default list" (which includes k8s.namespace.name and others), not
				// "extract nothing" - it must be explicitly emptied here so this instance only
				// ever claims annotation-derived keys, never built-in facts, before
				// resource/staticAttrs runs.
				"metadata": []string{},
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
			},
			"pod_association": k8sAttributesPodAssociation(),
		},
		k8sattributesFacts: map[string]any{
			"extract": map[string]any{
				"metadata": defaultK8Sattributes,
			},
			"pod_association": k8sAttributesPodAssociation(),
		},
		transform:      c.buildTransform(),
		transformPodIP: c.buildTransformPodIP(),
		batchTraces: &BatchConfig{
			SendBatchSize:    5000,
			SendBatchMaxSize: 5000,
			Timeout:          "60s",
		},
		batchMetrics: &BatchConfig{
			SendBatchSize:    3000,
			SendBatchMaxSize: 3000,
			Timeout:          "60s",
		},
		batchLogs: &BatchConfig{
			SendBatchSize:    1800,
			SendBatchMaxSize: 2000,
			Timeout:          "60s",
		},
		memoryLimiter: &MemoryLimiter{
			CheckInterval:         "1s",
			MemoryLimitPercentage: 70,
			MemorySpikePercentage: 30,
		},
	}

	if staticAttrs := c.buildStaticResourceAttributes(); staticAttrs != nil {
		processors[staticResourceAttrs] = staticAttrs
	}

	return processors
}

// k8sAttributesPodAssociation returns the pod_association config shared by both k8sattributes
// instances so each one can independently identify which pod a resource belongs to.
func k8sAttributesPodAssociation() []map[string]any {
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

// buildStaticResourceAttributes builds the resource/staticAttrs processor config, using action
// "insert" so it never overwrites a value already claimed by k8sattributesAnnotations (per-pod
// annotations must win), while still being able to claim a key before k8sattributesFacts runs
// (so the static DynaKube default wins over the built-in fact). See buildPipelineProcessors.
func (c *Config) buildStaticResourceAttributes() map[string]any {
	if len(c.resourceAttributes) == 0 {
		return nil
	}

	keys := make([]string, 0, len(c.resourceAttributes))
	for k := range c.resourceAttributes {
		keys = append(keys, k)
	}

	slices.Sort(keys)

	attributes := make([]map[string]any, 0, len(keys))
	for _, k := range keys {
		attributes = append(attributes, map[string]any{
			"key":    k,
			"value":  c.resourceAttributes[k],
			"action": "insert",
		})
	}

	return map[string]any{
		"attributes": attributes,
	}
}

func (c *Config) buildTransform() map[string]any {
	return map[string]any{
		"error_mode":        "ignore",
		"log_statements":    c.dynatraceTransformations(),
		"metric_statements": c.dynatraceTransformations(),
		"trace_statements":  c.dynatraceTransformations(),
	}
}

func (c *Config) buildTransformPodIP() map[string]any {
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

// dynatraceTransformations derives k8s.workload.name/kind and k8s.cluster.name from built-in
// facts using "== nil" guards, not just IsString(...): these run after resource/staticAttrs, so
// without the guard they would unconditionally overwrite a DynaKube resourceAttributes override
// with the built-in fact whenever the pod belongs to a real workload (i.e. almost always).
func (c *Config) dynatraceTransformations() []map[string]any {
	workloadNameFacts := []string{
		"k8s.statefulset.name",
		"k8s.replicaset.name",
		"k8s.job.name",
		"k8s.deployment.name",
		"k8s.daemonset.name",
		"k8s.cronjob.name",
	}

	nameStatements := make([]string, len(workloadNameFacts))
	for i, fact := range workloadNameFacts {
		nameStatements[i] = setValueIfPresentAndAbsent("k8s.workload.name", fact)
	}

	statements := slices.Concat([]string{
		`merge_maps(attributes, ParseJSON(attributes["metadata.dynatrace.com"]), "upsert") where IsMatch(attributes["metadata.dynatrace.com"], "^\\{")`,
		`delete_key(attributes, "metadata.dynatrace.com")`,
	}, nameStatements, []string{
		setLiteralIfPresentAndAbsent("k8s.workload.kind", "statefulset", "k8s.statefulset.name"),
		setLiteralIfPresentAndAbsent("k8s.workload.kind", "replicaset", "k8s.replicaset.name"),
		setLiteralIfPresentAndAbsent("k8s.workload.kind", "job", "k8s.job.name"),
		setLiteralIfPresentAndAbsent("k8s.workload.kind", "deployment", "k8s.deployment.name"),
		setLiteralIfPresentAndAbsent("k8s.workload.kind", "daemonset", "k8s.daemonset.name"),
		setLiteralIfPresentAndAbsent("k8s.workload.kind", "cronjob", "k8s.cronjob.name"),
		setLiteralIfAbsent("k8s.cluster.uid", "${env:K8S_CLUSTER_UID}"),
		setLiteralIfAbsent("k8s.cluster.name", "${env:K8S_CLUSTER_NAME}"),
		`set(attributes["dt.kubernetes.workload.name"], attributes["k8s.workload.name"])`,
		`set(attributes["dt.kubernetes.workload.kind"], attributes["k8s.workload.kind"])`,
		`set(attributes["dt.entity.kubernetes_cluster"], "${env:DT_ENTITY_KUBERNETES_CLUSTER}")`,
	}, deleteKeys(workloadNameFacts...))

	return []map[string]any{
		{
			"context":    "resource",
			"statements": statements,
		},
	}
}

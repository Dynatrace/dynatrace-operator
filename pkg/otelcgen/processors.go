package otelcgen

import (
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
	k8sattributes = component.MustNewID("k8sattributes")
	transform     = component.MustNewID("transform")
	batch         = component.MustNewType("batch")
	batchTraces   = component.NewIDWithName(batch, "traces")
	batchMetrics  = component.NewIDWithName(batch, "metrics")
	batchLogs     = component.NewIDWithName(batch, "logs")
	memoryLimiter = component.MustNewID("memory_limiter")

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
	return map[component.ID]component.Config{
		k8sattributes: map[string]any{
			"extract": map[string]any{
				"metadata": defaultK8Sattributes,
				"annotations": []map[string]any{
					{
						"from":      "pod",
						"key_regex": "metadata.dynatrace.com/(.*)",
						"tag_name":  "$$1",
					},
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
		},
		transform: c.buildTransform(),
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
}

func (c *Config) buildTransform() map[string]any {
	return map[string]any{
		"error_mode":        "ignore",
		"log_statements":    c.dynatraceTransformations(),
		"metric_statements": c.dynatraceTransformations(),
		"trace_statements":  c.dynatraceTransformations(),
	}
}

func (c *Config) dynatraceTransformations() []map[string]any {
	return []map[string]any{
		{
			"context": "resource",
			"statements": []string{
				"set(attributes[\"k8s.workload.name\"], attributes[\"k8s.statefulset.name\"]) where IsString(attributes[\"k8s.statefulset.name\"])",
				"set(attributes[\"k8s.workload.name\"], attributes[\"k8s.replicaset.name\"]) where IsString(attributes[\"k8s.replicaset.name\"])",
				"set(attributes[\"k8s.workload.name\"], attributes[\"k8s.job.name\"]) where IsString(attributes[\"k8s.job.name\"])",
				"set(attributes[\"k8s.workload.name\"], attributes[\"k8s.deployment.name\"]) where IsString(attributes[\"k8s.deployment.name\"])",
				"set(attributes[\"k8s.workload.name\"], attributes[\"k8s.daemonset.name\"]) where IsString(attributes[\"k8s.daemonset.name\"])",
				"set(attributes[\"k8s.workload.name\"], attributes[\"k8s.cronjob.name\"]) where IsString(attributes[\"k8s.cronjob.name\"])",
				"set(attributes[\"k8s.workload.kind\"], \"statefulset\") where IsString(attributes[\"k8s.statefulset.name\"])",
				"set(attributes[\"k8s.workload.kind\"], \"replicaset\") where IsString(attributes[\"k8s.replicaset.name\"])",
				"set(attributes[\"k8s.workload.kind\"], \"job\") where IsString(attributes[\"k8s.job.name\"])",
				"set(attributes[\"k8s.workload.kind\"], \"deployment\") where IsString(attributes[\"k8s.deployment.name\"])",
				"set(attributes[\"k8s.workload.kind\"], \"daemonset\") where IsString(attributes[\"k8s.daemonset.name\"])",
				"set(attributes[\"k8s.workload.kind\"], \"cronjob\") where IsString(attributes[\"k8s.cronjob.name\"])",
				"set(attributes[\"k8s.cluster.uid\"], \"${env:K8S_CLUSTER_UID}\") where attributes[\"k8s.cluster.uid\"] == nil",
				"set(attributes[\"k8s.cluster.name\"], \"${env:K8S_CLUSTER_NAME}\")",
				"set(attributes[\"dt.kubernetes.workload.name\"], attributes[\"k8s.workload.name\"])",
				"set(attributes[\"dt.kubernetes.workload.kind\"], attributes[\"k8s.workload.kind\"])",
				"set(attributes[\"dt.entity.kubernetes_cluster\"], \"${env:DT_ENTITY_KUBERNETES_CLUSTER}\")",
				"delete_key(attributes, \"k8s.statefulset.name\")",
				"delete_key(attributes, \"k8s.replicaset.name\")",
				"delete_key(attributes, \"k8s.job.name\")",
				"delete_key(attributes, \"k8s.deployment.name\")",
				"delete_key(attributes, \"k8s.daemonset.name\")",
				"delete_key(attributes, \"k8s.cronjob.name\")",
			},
		},
	}
}

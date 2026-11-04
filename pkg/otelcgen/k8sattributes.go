// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package otelcgen

import (
	"slices"

	"go.opentelemetry.io/collector/component"
)

// k8sAttributesOpts configures a k8s_attributes processor.
type k8sAttributesOpts struct {
	// Metadata lists the attributes to extract. A nil list lets the processor use its built-in
	// default list (which includes k8s.namespace.name and others), an empty list makes it extract
	// no metadata at all.
	Metadata []string

	// Annotations additionally extracts the metadata.dynatrace.com annotations of the pod.
	Annotations bool
}

// k8sAttributes builds the config of a k8s_attributes processor.
func k8sAttributes(opts k8sAttributesOpts) component.Config {
	extract := map[string]any{}

	if opts.Metadata != nil {
		extract["metadata"] = opts.Metadata
	}

	if opts.Annotations {
		extract["annotations"] = []map[string]any{
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
		}
	}

	return map[string]any{
		"extract":         extract,
		"pod_association": PodAssociation(),
	}
}

// k8sFactsMetadata returns the attributes a k8s_attributes processor extracts as facts about the pod.
// withUIDs adds the UIDs of the workloads owning the pod.
func k8sFactsMetadata(withUIDs bool) []string {
	metadata := []string{
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

	if withUIDs {
		metadata = append(metadata,
			"k8s.deployment.uid",
			"k8s.replicaset.uid",
			"k8s.statefulset.uid",
			"k8s.daemonset.uid",
			"k8s.cronjob.uid",
			"k8s.job.uid",
		)
	}

	return slices.Clone(metadata)
}

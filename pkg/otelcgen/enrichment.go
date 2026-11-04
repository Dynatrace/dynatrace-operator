// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package otelcgen

import (
	"go.opentelemetry.io/collector/pipeline"
)

// K8sEnrichmentOpts customizes the enrichment of the resource attributes with the metadata of Kubernetes.
// The order of the processors and of the statements that decides the precedence is not customizable.
type K8sEnrichmentOpts struct {
	// ResourceAttributes are static resource attributes, which are omitted if empty.
	ResourceAttributes map[string]string

	// WorkloadUIDs additionally extracts the UIDs of the workloads owning the pod and derives k8s.workload.uid.
	WorkloadUIDs bool

	// ExtraFactsMetadata are additional attributes the processor extracts as facts about the pod.
	ExtraFactsMetadata []string

	// Signals are the signals the transform is applied to. Empty means all signals.
	// The processors are registered with all pipelines the builder has.
	Signals []pipeline.Signal

	// ResourceStatements are additional statements of the resource context. They run after the workload attributes
	// are derived and before the facts about the owners are deleted, in the same group as these.
	ResourceStatements []string

	// Groups are additional statement groups, e.g. of the datapoint context, that run after the resource statements.
	Groups []map[string]any
}

// K8sEnrichment registers the processors that enrich the resource attributes with the metadata of Kubernetes.
// The resulting precedence is: metadata.dynatrace.com annotation > annotations > static resource attributes > facts.
//
//   - K8sAttributesAnnotationsID extracts the annotations first, as k8sattributesprocessor never overwrites an
//     attribute that is already set.
//   - StaticResourceAttrsID inserts the static resource attributes, if any, claiming their keys before the facts.
//   - K8sAttributesFactsID extracts the facts about the pod.
//   - TransformID merges the metadata.dynatrace.com blob first, with "upsert", as it already holds the resolved
//     precedence of the enrichment webhook. All statements that follow only fill gaps.
func (b *Builder) K8sEnrichment(opts K8sEnrichmentOpts) *Builder {
	b.Processor(K8sAttributesAnnotationsID, k8sAttributes(k8sAttributesOpts{Metadata: []string{}, Annotations: true}))

	if len(opts.ResourceAttributes) > 0 {
		b.Processor(StaticResourceAttrsID, staticResourceAttributes(opts.ResourceAttributes))
	}

	facts := append(k8sFactsMetadata(opts.WorkloadUIDs), opts.ExtraFactsMetadata...)
	b.Processor(K8sAttributesFactsID, k8sAttributes(k8sAttributesOpts{Metadata: facts}))

	signals := opts.Signals
	if len(signals) == 0 {
		signals = allSignals
	}

	groups := append(
		[]map[string]any{
			resourceStatements(
				mergeDynatraceAnnotations(),
				workloadFromOwners(opts.WorkloadUIDs),
				opts.ResourceStatements,
				deleteOwnerFacts(opts.WorkloadUIDs),
			),
		},
		opts.Groups...,
	)

	return b.Processor(TransformID, newTransform(signals, groups...))
}

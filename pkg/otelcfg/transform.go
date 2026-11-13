// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package otelcfg

import (
	"slices"

	"go.opentelemetry.io/collector/pipeline"
)

// workloadOwners are the kinds of workloads that own a pod, sorted in order of highest in the
// chain, e.g. a deployment-owned pod shouldn't use its replicaset.
var workloadOwners = []string{"daemonset", "statefulset", "deployment", "replicaset", "cronjob", "job"}

// mergeDynatraceAnnotations returns the statements that unpack the metadata.dynatrace.com JSON
// blob (written by the enrichment webhook, which already resolves the full precedence chain) into
// the resource attributes. They overwrite existing attributes, so they must come first: every
// statement after them only fills gaps, and therefore never replaces a value from the blob.
func mergeDynatraceAnnotations() []string {
	return []string{
		`merge_maps(attributes, ParseJSON(attributes["metadata.dynatrace.com"]), "upsert") where IsMatch(attributes["metadata.dynatrace.com"], "^\\{")`,
		`delete_key(attributes, "metadata.dynatrace.com")`,
	}
}

// workloadFromOwners returns the statements that derive k8s.workload.name and k8s.workload.kind,
// and k8s.workload.uid if withUIDs is set, from the facts about the owners of the pod. They use
// "== nil" guards, not just IsString(...): they run after resource/staticAttrs, so without the
// guard they would unconditionally overwrite a DynaKube resourceAttributes override with the
// built-in fact whenever the pod belongs to a real workload (i.e. almost always).
func workloadFromOwners(withUIDs bool) []string {
	statements := make([]string, 0, len(workloadOwners))

	for _, owner := range workloadOwners {
		statements = append(statements, setValueIfPresentAndAbsent("k8s.workload.name", "k8s."+owner+".name"))
	}

	for _, owner := range workloadOwners {
		statements = append(statements, setLiteralIfPresentAndAbsent("k8s.workload.kind", owner, "k8s."+owner+".name"))
	}

	if withUIDs {
		for _, owner := range workloadOwners {
			statements = append(statements, setValueIfPresentAndAbsent("k8s.workload.uid", "k8s."+owner+".uid"))
		}
	}

	return statements
}

// deleteOwnerFacts returns the statements that delete the facts about the owners of the pod, which
// are only needed to derive the workload attributes.
func deleteOwnerFacts(withUIDs bool) []string {
	keys := make([]string, 0, len(workloadOwners))

	for _, owner := range workloadOwners {
		keys = append(keys, "k8s."+owner+".name")
	}

	if withUIDs {
		for _, owner := range workloadOwners {
			keys = append(keys, "k8s."+owner+".uid")
		}
	}

	return DeleteKeys(keys...)
}

// resourceStatements returns a statement group of the resource context.
func resourceStatements(statements ...[]string) map[string]any {
	return map[string]any{
		"context":    "resource",
		"statements": slices.Concat(statements...),
	}
}

// newTransform builds the config of a transform processor that runs the statement groups on the
// given signals. error_mode is always set, so a statement that fails to run is logged and skipped
// regardless of the default of the collector.
func newTransform(signals []pipeline.Signal, groups ...map[string]any) map[string]any {
	cfg := map[string]any{"error_mode": "ignore"}

	for _, signal := range signals {
		switch signal {
		case pipeline.SignalTraces:
			cfg["trace_statements"] = groups
		case pipeline.SignalMetrics:
			cfg["metric_statements"] = groups
		case pipeline.SignalLogs:
			cfg["log_statements"] = groups
		}
	}

	return cfg
}

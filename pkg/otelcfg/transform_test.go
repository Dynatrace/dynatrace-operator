// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package otelcfg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pipeline"
)

func TestMergeDynatraceAnnotations(t *testing.T) {
	statements := mergeDynatraceAnnotations()

	require.Len(t, statements, 2)
	assert.Contains(t, statements[0], `merge_maps(attributes, ParseJSON(attributes["metadata.dynatrace.com"]), "upsert")`)
	assert.Equal(t, `delete_key(attributes, "metadata.dynatrace.com")`, statements[1])
}

func TestWorkloadFromOwners(t *testing.T) {
	without := workloadFromOwners(false)
	with := workloadFromOwners(true)

	t.Run("name and kind are derived for every owner, highest in the chain first", func(t *testing.T) {
		require.Len(t, without, 12)
		assert.Equal(t, setValueIfPresentAndAbsent("k8s.workload.name", "k8s.daemonset.name"), without[0])
		assert.Equal(t, setValueIfPresentAndAbsent("k8s.workload.name", "k8s.job.name"), without[5])
		assert.Equal(t, setLiteralIfPresentAndAbsent("k8s.workload.kind", "daemonset", "k8s.daemonset.name"), without[6])
		assert.Equal(t, setLiteralIfPresentAndAbsent("k8s.workload.kind", "job", "k8s.job.name"), without[11])
	})

	t.Run("deployment is before replicaset", func(t *testing.T) {
		assert.Less(t,
			indexOf(t, without, setValueIfPresentAndAbsent("k8s.workload.name", "k8s.deployment.name")),
			indexOf(t, without, setValueIfPresentAndAbsent("k8s.workload.name", "k8s.replicaset.name")),
		)
	})

	t.Run("the uid is only derived if requested", func(t *testing.T) {
		require.Len(t, with, 18)
		assert.Equal(t, without, with[:12])

		for _, statement := range without {
			assert.NotContains(t, statement, "k8s.workload.uid")
		}

		assert.Equal(t, setValueIfPresentAndAbsent("k8s.workload.uid", "k8s.daemonset.uid"), with[12])
	})
}

func TestDeleteOwnerFacts(t *testing.T) {
	assert.Len(t, deleteOwnerFacts(false), 6)
	assert.Contains(t, deleteOwnerFacts(false), `delete_key(attributes, "k8s.deployment.name")`)
	assert.NotContains(t, deleteOwnerFacts(false), `delete_key(attributes, "k8s.deployment.uid")`)

	assert.Len(t, deleteOwnerFacts(true), 12)
	assert.Contains(t, deleteOwnerFacts(true), `delete_key(attributes, "k8s.deployment.uid")`)
}

func TestTransform(t *testing.T) {
	group := resourceStatements([]string{"a"}, []string{"b"})

	assert.Equal(t, map[string]any{"context": "resource", "statements": []string{"a", "b"}}, group)

	t.Run("only the requested signals have statements", func(t *testing.T) {
		cfg := newTransform([]pipeline.Signal{pipeline.SignalMetrics}, group)

		assert.Equal(t, map[string]any{
			"error_mode":        "ignore",
			"metric_statements": []map[string]any{group},
		}, cfg)
	})

	t.Run("all signals", func(t *testing.T) {
		cfg := newTransform([]pipeline.Signal{pipeline.SignalTraces, pipeline.SignalMetrics, pipeline.SignalLogs}, group)

		assert.Contains(t, cfg, "trace_statements")
		assert.Contains(t, cfg, "metric_statements")
		assert.Contains(t, cfg, "log_statements")
	})
}

func indexOf(t *testing.T, statements []string, statement string) int {
	t.Helper()

	for i, s := range statements {
		if s == statement {
			return i
		}
	}

	require.Failf(t, "statement not found", "%s", statement)

	return -1
}

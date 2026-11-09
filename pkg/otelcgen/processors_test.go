// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package otelcgen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
)

// enrichmentProcessors returns the processors that are registered by K8sEnrichment.
func enrichmentProcessors() map[component.ID]component.Config {
	return NewBuilder().K8sEnrichment(K8sEnrichmentOpts{}).Build().Processors
}

func TestK8sAttributesJSONAnnotation(t *testing.T) {
	processors := enrichmentProcessors()

	k8sAttr, ok := processors[K8sAttributesAnnotationsID].(map[string]any)
	require.True(t, ok)

	extract, ok := k8sAttr["extract"].(map[string]any)
	require.True(t, ok)

	// k8sattributesprocessor treats an omitted "metadata" field as "use its built-in default
	// list" rather than "extract nothing", so it must be explicitly emptied here - otherwise
	// this instance would also claim built-in facts (e.g. k8s.namespace.name) before
	// resource/staticAttrs gets a chance to run.
	assert.Equal(t, []string{}, extract["metadata"])

	annotations, ok := extract["annotations"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, annotations, 2)

	t.Run("regex rule kept", func(t *testing.T) {
		rule := annotations[0]
		assert.Equal(t, "pod", rule["from"])
		assert.Equal(t, "metadata.dynatrace.com/(.*)", rule["key_regex"])
		assert.Equal(t, "$$1", rule["tag_name"])
		assert.NotContains(t, rule, "key")
	})

	t.Run("exact-key rule added", func(t *testing.T) {
		rule := annotations[1]
		assert.Equal(t, "pod", rule["from"])
		assert.Equal(t, "metadata.dynatrace.com", rule["key"])
		assert.Equal(t, "metadata.dynatrace.com", rule["tag_name"])
		assert.NotContains(t, rule, "key_regex")
	})
}

func TestK8sAttributesFacts(t *testing.T) {
	processors := enrichmentProcessors()

	k8sAttr, ok := processors[K8sAttributesFactsID].(map[string]any)
	require.True(t, ok)

	extract, ok := k8sAttr["extract"].(map[string]any)
	require.True(t, ok)

	assert.NotContains(t, extract, "annotations")
	assert.Equal(t, k8sFactsMetadata(false), extract["metadata"])
}

func TestMemoryLimiter(t *testing.T) {
	assert.Equal(
		t,
		map[string]any{"check_interval": "1s", "limit_percentage": uint32(95), "spike_limit_percentage": uint32(5)},
		MemoryLimiter(MemoryLimiterOpts{LimitPercentage: 95, SpikePercentage: 5}),
	)
}

func TestPodAssociation(t *testing.T) {
	got := PodAssociation()

	require.Len(t, got, 4)
	assert.Equal(t, []map[string]any{{"from": "connection"}}, got[3]["sources"])
}

func TestK8sAttributes(t *testing.T) {
	t.Run("nil metadata is omitted so the processor uses its defaults", func(t *testing.T) {
		cfg, ok := k8sAttributes(k8sAttributesOpts{}).(map[string]any)
		require.True(t, ok)

		assert.Equal(t, map[string]any{}, cfg["extract"])
		assert.Equal(t, PodAssociation(), cfg["pod_association"])
	})

	t.Run("empty metadata is kept so the processor extracts nothing", func(t *testing.T) {
		cfg, ok := k8sAttributes(k8sAttributesOpts{Metadata: []string{}}).(map[string]any)
		require.True(t, ok)

		assert.Equal(t, map[string]any{"metadata": []string{}}, cfg["extract"])
	})

	t.Run("annotations are only extracted when requested", func(t *testing.T) {
		with, ok := k8sAttributes(k8sAttributesOpts{Annotations: true}).(map[string]any)
		require.True(t, ok)
		assert.Contains(t, with["extract"], "annotations")

		without, ok := k8sAttributes(k8sAttributesOpts{}).(map[string]any)
		require.True(t, ok)
		assert.NotContains(t, without["extract"], "annotations")
	})
}

func TestK8sFactsMetadata(t *testing.T) {
	without := k8sFactsMetadata(false)
	with := k8sFactsMetadata(true)

	assert.NotContains(t, without, "k8s.deployment.uid")
	assert.Equal(t, without, with[:len(without)], "UIDs are only appended")
	assert.ElementsMatch(t, []string{
		"k8s.deployment.uid", "k8s.replicaset.uid", "k8s.statefulset.uid",
		"k8s.daemonset.uid", "k8s.cronjob.uid", "k8s.job.uid",
	}, with[len(without):])

	// callers may modify the result
	without[0] = "changed"
	assert.NotEqual(t, "changed", k8sFactsMetadata(false)[0])
}

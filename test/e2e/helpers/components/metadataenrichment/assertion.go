// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package metadataenrichment

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8spod"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/shell"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
)

const (
	MetadataFile       = "/var/lib/dynatrace/enrichment/dt_metadata.json"
	PropertiesFile     = "/var/lib/dynatrace/enrichment/dt_metadata.properties"
	NodePropertiesFile = "/var/lib/dynatrace/enrichment/dt_node_metadata.properties"
)

type Metadata struct {
	WorkloadKind string `json:"k8s.workload.kind,omitempty"`
	WorkloadName string `json:"k8s.workload.name,omitempty"`

	// deprecated fields, should exist only if enable-attributes-dt.kubernetes feature flag is enabled
	DTWorkloadKind string `json:"dt.kubernetes.workload.kind,omitempty"`
	DTWorkloadName string `json:"dt.kubernetes.workload.name,omitempty"`
}

func GetMetadataJSONFromPod(ctx context.Context, t *testing.T, resource *resources.Resources, enrichedPod corev1.Pod) Metadata {
	content := readMetadataFile(ctx, t, resource, enrichedPod, MetadataFile)

	var enrichmentMetadata Metadata
	err := json.Unmarshal(content, &enrichmentMetadata)
	require.NoError(t, err)

	return enrichmentMetadata
}

func GetRawMetadataFromPod(ctx context.Context, t *testing.T, resource *resources.Resources, enrichedPod corev1.Pod) []byte {
	return readMetadataFile(ctx, t, resource, enrichedPod, MetadataFile)
}

func GetMetadataMapFromPod(ctx context.Context, t *testing.T, resource *resources.Resources, enrichedPod corev1.Pod) map[string]string {
	var metadata map[string]string
	require.NoError(t, json.Unmarshal(GetRawMetadataFromPod(ctx, t, resource, enrichedPod), &metadata))

	return metadata
}

func GetMetadataPropertiesFromPod(ctx context.Context, t *testing.T, resource *resources.Resources, enrichedPod corev1.Pod) map[string]string {
	properties := readMetadataFile(ctx, t, resource, enrichedPod, PropertiesFile)

	return parseProperties(string(properties))
}

func GetNodeMetadataPropertiesFromPod(ctx context.Context, t *testing.T, resource *resources.Resources, enrichedPod corev1.Pod) map[string]string {
	properties := readMetadataFile(ctx, t, resource, enrichedPod, NodePropertiesFile)

	return parseProperties(string(properties))
}

func readMetadataFile(ctx context.Context, t *testing.T, resource *resources.Resources, enrichedPod corev1.Pod, path string) []byte {
	require.NotEmpty(t, enrichedPod.Spec.Containers)
	enrichedContainer := enrichedPod.Spec.Containers[0].Name
	readMetadataCommand := shell.ReadFile(path)
	result, err := k8spod.Exec(ctx, resource, enrichedPod, enrichedContainer, readMetadataCommand...)

	require.NoError(t, err)

	assert.Zero(t, result.StdErr.Len())
	assert.NotEmpty(t, result.StdOut)

	return result.StdOut.Bytes()
}

func parseProperties(text string) map[string]string {
	m := make(map[string]string)

	for line := range strings.Lines(text) {
		l := strings.TrimSpace(line)
		if l == "" {
			continue
		}

		key, value, found := strings.Cut(l, "=")
		if !found {
			continue
		}

		m[key] = value
	}

	return m
}

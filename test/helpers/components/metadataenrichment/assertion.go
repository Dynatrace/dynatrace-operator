//go:build e2e

package metadataenrichment

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubernetes/objects/k8spod"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/shell"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
)

const (
	MetadataFile = "/var/lib/dynatrace/enrichment/dt_metadata.json"
)

type Metadata struct {
	WorkloadKind string `json:"k8s.workload.kind,omitempty"`
	WorkloadName string `json:"k8s.workload.name,omitempty"`

	// deprecated fields, should exist only if enable-attributes-dt.kubernetes feature flag is enabled
	DtWorkloadKind string `json:"dt.kubernetes.workload.kind,omitempty"`
	DtWorkloadName string `json:"dt.kubernetes.workload.name,omitempty"`
}

func GetMetadataFromPod(ctx context.Context, t *testing.T, resource *resources.Resources, enrichedPod corev1.Pod) Metadata {
	require.NotEmpty(t, enrichedPod.Spec.Containers)
	enrichedContainer := enrichedPod.Spec.Containers[0].Name
	readMetadataCommand := shell.ReadFile(MetadataFile)
	result, err := k8spod.Exec(ctx, resource, enrichedPod, enrichedContainer, readMetadataCommand...)

	require.NoError(t, err)

	assert.Zero(t, result.StdErr.Len())
	assert.NotEmpty(t, result.StdOut)

	var enrichmentMetadata Metadata
	err = json.Unmarshal(result.StdOut.Bytes(), &enrichmentMetadata)

	require.NoError(t, err)

	return enrichmentMetadata
}

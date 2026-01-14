//go:build e2e

package cloudnative

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sample"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/shell"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	metadataFile = "/var/lib/dynatrace/enrichment/dt_metadata.json"
)

type metadata struct {
	WorkloadKind string `json:"dt.kubernetes.workload.kind,omitempty"`
	WorkloadName string `json:"dt.kubernetes.workload.name,omitempty"`
}

func AssessMetadataEnrichment(builder *features.FeatureBuilder, sampleApp *sample.App) {
	builder.Assess("metadata enrichment enabled", checkMetadataEnrichment(sampleApp))
}

func checkMetadataEnrichment(sampleApp *sample.App) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		kubeResources := envConfig.Client().Resources()

		// Use the logic to verify file presence and content
		// We expect "deployment" kind for the sample app
		deploymentName := sampleApp.Name()

		pods := sampleApp.GetPods(ctx, t, kubeResources)
		require.NotEmpty(t, pods.Items)

		for _, podItem := range pods.Items {
			enrichmentMetadata := getMetadataEnrichmentMetadataFromPod(ctx, t, kubeResources, podItem)
			assert.Equal(t, "deployment", enrichmentMetadata.WorkloadKind)
			assert.Equal(t, deploymentName, enrichmentMetadata.WorkloadName)
		}

		return ctx
	}
}

func getMetadataEnrichmentMetadataFromPod(ctx context.Context, t *testing.T, resource *resources.Resources, enrichedPod corev1.Pod) metadata {
	require.NotEmpty(t, enrichedPod.Spec.Containers)
	enrichedContainer := enrichedPod.Spec.Containers[0].Name
	readMetadataCommand := shell.ReadFile(metadataFile)
	result, err := pod.Exec(ctx, resource, enrichedPod, enrichedContainer, readMetadataCommand...)

	require.NoError(t, err)

	assert.Zero(t, result.StdErr.Len())
	assert.NotEmpty(t, result.StdOut)

	var enrichmentMetadata metadata
	err = json.Unmarshal(result.StdOut.Bytes(), &enrichmentMetadata)

	require.NoError(t, err)

	return enrichmentMetadata
}

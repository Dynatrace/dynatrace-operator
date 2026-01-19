//go:build e2e

package cloudnative

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sample"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func AssessMetadataEnrichment(builder *features.FeatureBuilder, sampleApp *sample.App) {
	builder.Assess("check metadata enrichment", checkMetadataEnrichment(sampleApp))
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
			enrichmentMetadata := metadataenrichment.GetMetadataFromPod(ctx, t, kubeResources, podItem)
			assert.Equal(t, "deployment", enrichmentMetadata.WorkloadKind)
			assert.Equal(t, deploymentName, enrichmentMetadata.WorkloadName)
		}

		return ctx
	}
}

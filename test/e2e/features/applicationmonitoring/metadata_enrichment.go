// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package applicationmonitoring

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	maputil "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	dynakubeComponents "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/sample"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/tenant"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

// MetadataEnrichment verifies that a pod injected with both OneAgent and metadata enrichment ends
// up with a real dt_metadata.json file that contains the expected workload and (deprecated)
// dt.kubernetes.* attributes. Reading that file requires an actual running pod; the mutation logic
// that decides which flags/attributes get added for which annotation/namespace-selector combination
// is covered by envtest (webhook_integration_test.go) and unit tests closer to the webhook code.
func MetadataEnrichment(t *testing.T) features.Feature {
	builder := features.New("metadata-enrichment")
	secretConfig := tenant.GetSingleTenantSecret(t)
	testDynakube := *dynakubeComponents.New(
		dynakubeComponents.WithAPIURL(secretConfig.APIURL),
		dynakubeComponents.WithMetadataEnrichment(),
		dynakubeComponents.WithApplicationMonitoringSpec(&oneagent.ApplicationMonitoringSpec{}),
		dynakubeComponents.WithNameBasedMetadataEnrichmentNamespaceSelector(),
		dynakubeComponents.WithNameBasedOneAgentNamespaceSelector(),
		dynakubeComponents.WithAnnotations(map[string]string{exp.EnrichmentEnableAttributesDTKubernetes: "true"}),
	)

	injectEverythingLabels := maputil.MergeMap(
		testDynakube.OneAgent().GetNamespaceSelector().MatchLabels,
		testDynakube.MetadataEnrichment().GetNamespaceSelector().MatchLabels,
	)

	sampleApp := sample.NewApp(t, &testDynakube,
		sample.WithName("pod-with-dt-attributes"),
		sample.WithNamespaceLabels(injectEverythingLabels),
	)

	dynakubeComponents.Install(builder, &secretConfig, testDynakube)
	builder.Assess("Installing sample app", sampleApp.Install())
	builder.Assess("Checking dt_metadata.json content", assessMetadataEnrichmentHasDeprecatedAttributes(sampleApp))
	builder.WithTeardown("Uninstalling sample app", sampleApp.Uninstall())

	return builder.Feature()
}

func assessMetadataEnrichmentHasDeprecatedAttributes(samplePod *sample.App) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		testPod := samplePod.ListPods(ctx, t, envConfig.Client().Resources()).Items[0]
		enrichmentMetadata := metadataenrichment.GetMetadataJSONFromPod(ctx, t, envConfig.Client().Resources(), testPod)

		assert.Equal(t, "pod", enrichmentMetadata.DTWorkloadKind)
		assert.Equal(t, testPod.Name, enrichmentMetadata.DTWorkloadName)

		assert.Equal(t, "pod", enrichmentMetadata.WorkloadKind)
		assert.Equal(t, testPod.Name, enrichmentMetadata.WorkloadName)

		return ctx
	}
}

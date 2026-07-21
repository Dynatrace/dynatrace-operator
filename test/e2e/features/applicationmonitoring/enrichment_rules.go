//go:build e2e

package applicationmonitoring

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	maputil "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	dynakubeComponents "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/dynakube"
	enrichment "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/sample"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/tenant"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const enrichmentLabelValue = "e2e-enrichment-test"

// EnrichmentRules verifies that enrichment rules created on the tenant are read and stored in dk.Status.MetadataEnrichment.Rules,
// and that the mapped attribute appears in dt_metadata.json inside an injected pod.
func EnrichmentRules(t *testing.T) features.Feature {
	builder := features.New("enrichment-rules")
	secretConfig := tenant.GetSingleTenantSecret(t)

	expectedRule := metadataenrichment.Rule{
		Type:   metadataenrichment.LabelRule,
		Source: "e2e-test-label",
		Target: "dt.cost.product",
	}

	// on phase 3 only the new schema is supported, which uses the "K8S_NAMESPACE_LABEL" rule type
	if tenant.UsePhase3Tenant() {
		expectedRule.Type = metadataenrichment.K8sNamespaceLabelRule
	}

	// Setup: pre-create the Kubernetes Cluster MEID on the tenant so the rule can be
	// scoped directly to the cluster without waiting for DynaKube reconciliation.
	// Then clean any leftover rules from previous runs before creating the test rule.
	builder.Setup(enrichment.EnsureKubernetesClusterMEID(secretConfig))
	builder.Setup(enrichment.DeleteEnrichmentRulesFromTenant(secretConfig))

	// Be aware that this requires additional permissions on the service user group if platform token is used
	// Add a new policy to your service user group that allows write access to 'ingest.enrichment.config'
	builder.Setup(enrichment.CreateEnrichmentRuleOnTenant(secretConfig, expectedRule))

	testDynakube := dynakubeComponents.New(
		dynakubeComponents.WithAPIURL(secretConfig.APIURL),
		dynakubeComponents.WithMetadataEnrichment(),
		dynakubeComponents.WithNameBasedMetadataEnrichmentNamespaceSelector(),
	)
	dynakubeComponents.Install(builder, &secretConfig, *testDynakube)

	builder.Assess("enrichment rule is stored in DynaKube status",
		enrichment.CheckEnrichmentRuleInDynaKubeStatus(testDynakube, expectedRule))

	sampleApp := sample.NewApp(t, testDynakube,
		sample.WithName("enrichment-rule-app"),
		sample.WithNamespaceLabels(maputil.MergeMap(
			testDynakube.MetadataEnrichment().GetNamespaceSelector().MatchLabels,
			map[string]string{expectedRule.Source: enrichmentLabelValue},
		)),
	)

	builder.Assess("install sample app", sampleApp.Install())
	builder.Assess("enrichment rule attribute is present in dt_metadata.json",
		checkEnrichmentAttributeInMetadataFile(sampleApp, expectedRule.Target, enrichmentLabelValue))

	builder.WithTeardown("uninstall sample app", sampleApp.Uninstall())
	builder.WithTeardown("delete enrichment rules from tenant",
		enrichment.DeleteEnrichmentRulesFromTenant(secretConfig))

	return builder.Feature()
}

func checkEnrichmentAttributeInMetadataFile(app *sample.App, key, value string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		pod := app.GetPod(ctx, t, envConfig.Client().Resources())
		metadata := enrichment.GetMetadataMapFromPod(ctx, t, envConfig.Client().Resources(), pod)

		assert.Equal(t, value, metadata[key])

		return ctx
	}
}

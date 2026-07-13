//go:build e2e

package applicationmonitoring

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	dynakubeComponents "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/tenant"
	componentEnrichment "github.com/Dynatrace/dynatrace-operator/test/helpers/components/metadataenrichment"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

// EnrichmentRules verifies that enrichment rules created on the tenant are read and stored in dk.Status.MetadataEnrichment.Rules.
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
	builder.Setup(componentEnrichment.EnsureKubernetesClusterMEID(secretConfig))
	builder.Setup(componentEnrichment.DeleteEnrichmentRulesFromTenant(secretConfig))

	// Be aware that this requires additional permissions on the service user group if platform token is used
	// Add a new policy to your service user group that allows write access to 'ingest.enrichment.config'
	builder.Setup(componentEnrichment.CreateEnrichmentRuleOnTenant(secretConfig, expectedRule))

	testDynakube := dynakubeComponents.New(
		dynakubeComponents.WithAPIURL(secretConfig.APIURL),
		dynakubeComponents.WithMetadataEnrichment(),
		dynakubeComponents.WithApplicationMonitoringSpec(&oneagent.ApplicationMonitoringSpec{}),
	)
	dynakubeComponents.Install(builder, &secretConfig, *testDynakube)

	builder.Assess("enrichment rule is stored in DynaKube status",
		componentEnrichment.CheckEnrichmentRuleInDynaKubeStatus(testDynakube, expectedRule))

	builder.WithTeardown("delete enrichment rules from tenant",
		componentEnrichment.DeleteEnrichmentRulesFromTenant(secretConfig))

	return builder.Feature()
}

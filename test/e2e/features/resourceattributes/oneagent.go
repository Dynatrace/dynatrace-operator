//go:build e2e

package resourceattributes

import (
	"testing"

	maputil "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/features/cloudnative"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/features/consts"
	dynakubeComponents "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8sdaemonset"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/tenant"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func OneAgent(t *testing.T) features.Feature {
	builder := features.New("resource-attributes-oneagent")
	secretConfig := tenant.GetSingleTenantSecret(t)
	ns := "resource-attributes-oneagent"

	testDynakube := *dynakubeComponents.New(
		dynakubeComponents.WithAPIURL(secretConfig.APIURL),
		dynakubeComponents.WithCloudNativeSpec(cloudnative.DefaultCloudNativeSpec()),
		dynakubeComponents.WithMetadataEnrichment(),
		dynakubeComponents.WithNameBasedOneAgentNamespaceSelector(),
		dynakubeComponents.WithNameBasedMetadataEnrichmentNamespaceSelector(),
		dynakubeComponents.WithResourceAttributes(globalAttrs),
		dynakubeComponents.WithOneAgentAdditionalResourceAttributes(oneAgentAdditional),
		// to be removed before merge
		dynakubeComponents.WithAnnotations(map[string]string{"feature.dynatrace.com/use-public-registry": "true"}),
		dynakubeComponents.WithCustomPullSecret(consts.DevRegistryPullSecretName),
	)

	injectEverythingLabels := maputil.MergeMap(
		testDynakube.OneAgent().GetNamespaceSelector().MatchLabels,
		testDynakube.MetadataEnrichment().GetNamespaceSelector().MatchLabels,
	)

	sampleApp := newSampleApp(t, &testDynakube, ns, injectEverythingLabels)

	dynakubeComponents.Install(builder, &secretConfig, testDynakube)
	builder.Assess("OneAgent DaemonSet is ready", k8sdaemonset.IsReady(testDynakube.OneAgent().GetDaemonsetName(), testDynakube.Namespace))

	builder.Assess("OneAgent dt_node_metadata.properties contains merged OneAgent resource attributes", assessDTNodeMetadataProperties(testDynakube, expectedOneAgent))

	installSampleApp(builder, sampleApp)

	builder.Assess("initcontainer contains args with additionalAttributes", assessInitContainerArgs(sampleApp, expectedOneAgent))
	builder.Assess("dt_metadata.json and dt_metadata.properties contains merged OneAgent resource attributes", assessDTMetadataFiles(sampleApp, expectedOneAgent))
	builder.Assess("metadata.dynatrace.com JSON annotation contains merged OneAgent resource attributes", assessPodMetadataJSONAnnotation(sampleApp, expectedOneAgent))
	builder.Assess("metadata.dynatrace.com/* individual annotations contain merged OneAgent resource attributes", assessPodIndividualAnnotations(sampleApp, expectedOneAgent))

	uninstallSampleApp(builder, sampleApp)

	return builder.Feature()
}

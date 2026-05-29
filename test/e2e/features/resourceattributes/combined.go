//go:build e2e

package resourceattributes

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/otlp"
	activegateconsts "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	maputil "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/features/cloudnative"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/activegate"
	dynakubeComponents "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8sdaemonset"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func Combined(t *testing.T) features.Feature {
	builder := features.New("resource-attributes-combined")
	secretConfig := tenant.GetSingleTenantSecret(t)
	ns := "resource-attributes-combined"

	// expectedOTLPInAll is the effective OTEL_RESOURCE_ATTRIBUTES content
	// when OA and OTLP are both injected into the same pod.
	expectedOTLPInAll := map[string]string{
		"deployment.environment": "otlp-env", // OTLP annotation wins over OA dynakube attr
		"service.namespace":      "global-ns",
		"global.only.key":        "global-only-value",
		"otlp.only.key":          "otlp-only-value",
	}

	testDynakube := *dynakubeComponents.New(
		dynakubeComponents.WithAPIURL(secretConfig.APIURL),
		dynakubeComponents.WithCloudNativeSpec(cloudnative.DefaultCloudNativeSpec()),
		dynakubeComponents.WithMetadataEnrichment(),
		dynakubeComponents.WithActiveGate(),
		dynakubeComponents.WithLogMonitoring(),
		dynakubeComponents.WithLogMonitoringImageRef(t),
		dynakubeComponents.WithNameBasedOneAgentNamespaceSelector(),
		dynakubeComponents.WithNameBasedMetadataEnrichmentNamespaceSelector(),
		dynakubeComponents.WithNameBasedOTLPNamespaceSelector(),
		dynakubeComponents.WithOTLPSignals(otlp.SignalConfiguration{
			Metrics: &otlp.MetricsSignal{},
			Logs:    &otlp.LogsSignal{},
			Traces:  &otlp.TracesSignal{},
		}),
		dynakubeComponents.WithResourceAttributes(globalAttrs),
		dynakubeComponents.WithOneAgentAdditionalResourceAttributes(oneAgentAdditional),
		dynakubeComponents.WithOTLPAdditionalResourceAttributes(otlpAdditional),
		devRegistryOptions(),
	)

	injectEverythingLabels := maputil.MergeMap(
		testDynakube.OneAgent().GetNamespaceSelector().MatchLabels,
		testDynakube.MetadataEnrichment().GetNamespaceSelector().MatchLabels,
		testDynakube.Spec.OTLPExporterConfiguration.NamespaceSelector.MatchLabels,
	)

	sampleApp := newSampleApp(t, &testDynakube, ns, injectEverythingLabels)

	dynakubeComponents.Install(builder, &secretConfig, testDynakube)
	builder.Assess("OneAgent DaemonSet is ready", k8sdaemonset.IsReady(testDynakube.OneAgent().GetDaemonsetName(), testDynakube.Namespace))
	builder.Assess("ActiveGate is running", activegate.CheckContainer(&testDynakube))

	builder.Assess("OneAgent dt_node_metadata.properties contains merged OneAgent resource attributes", assessDTNodeMetadataProperties(testDynakube, sampleApp, expectedOneAgent))
	builder.Assess("ActiveGate deployment.properties ConfigMap contains global resource attributes", assessActiveGateDeploymentProperties(testDynakube, globalAttrs))

	installSampleApp(builder, sampleApp)

	builder.Assess("initcontainer contains args with additionalAttributes", assessInitContainerArgs(sampleApp, expectedOneAgent))
	builder.Assess("dt_metadata.json and dt_metadata.properties contains merged OneAgent resource attributes", assessDTMetadataFiles(testDynakube, sampleApp, expectedOneAgent))
	builder.Assess("OTEL_RESOURCE_ATTRIBUTES contains merged OTLP resource attributes (OA wins shared keys)", assessOTLPInjectionAttributes(testDynakube, sampleApp, expectedOTLPInAll))
	builder.Assess("metadata.dynatrace.com JSON annotation contains merged OneAgent resource attributes", assessPodMetadataJSONAnnotation(sampleApp, expectedOneAgent))
	builder.Assess("metadata.dynatrace.com/* individual annotations contain merged OneAgent resource attributes", assessPodIndividualAnnotations(sampleApp, expectedOneAgent))

	uninstallSampleApp(builder, sampleApp)

	return builder.Feature()
}

func assessActiveGateDeploymentProperties(dk dynakube.DynaKube, expected map[string]string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		var cm corev1.ConfigMap
		err := envConfig.Client().Resources().Get(ctx, dk.ActiveGate().GetDeploymentPropertiesConfigMapName(), dk.Namespace, &cm)
		require.NoError(t, err)

		content, ok := cm.Data[activegateconsts.DeploymentPropertiesFileName]
		require.Truef(t, ok, "%s key missing in ConfigMap %s", activegateconsts.DeploymentPropertiesFileName, cm.Name)
		require.Contains(t, content, "[resource_attributes]",
			"ConfigMap %s is missing the [resource_attributes] section", cm.Name)

		for k, v := range expected {
			assert.Containsf(t, content, k+" = "+v, "deployment.properties missing %s = %s", k, v)
		}

		return ctx
	}
}

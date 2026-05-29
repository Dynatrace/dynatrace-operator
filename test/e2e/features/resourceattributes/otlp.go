//go:build e2e

package resourceattributes

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/otlp"
	dynakubeComponents "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/tenant"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func OTLPExporterConfig(t *testing.T) features.Feature {
	builder := features.New("resource-attributes-otlp")
	secretConfig := tenant.GetSingleTenantSecret(t)
	ns := "resource-attributes-otlp"

	testDynakube := *dynakubeComponents.New(
		dynakubeComponents.WithAPIURL(secretConfig.APIURL),
		dynakubeComponents.WithNameBasedOTLPNamespaceSelector(),
		dynakubeComponents.WithOTLPSignals(otlp.SignalConfiguration{
			Metrics: &otlp.MetricsSignal{},
			Logs:    &otlp.LogsSignal{},
			Traces:  &otlp.TracesSignal{},
		}),
		dynakubeComponents.WithResourceAttributes(globalAttrs),
		dynakubeComponents.WithOTLPAdditionalResourceAttributes(otlpAdditional),
		devRegistryOptions(),
	)

	sampleApp := newSampleApp(t, &testDynakube, ns, testDynakube.Spec.OTLPExporterConfiguration.NamespaceSelector.MatchLabels)

	dynakubeComponents.Install(builder, &secretConfig, testDynakube)
	installSampleApp(builder, sampleApp)

	builder.Assess("OTEL_RESOURCE_ATTRIBUTES contains merged OTLP resource attributes", assessOTLPInjectionAttributes(testDynakube, sampleApp, expectedOTLP))
	// The OTLP resource-attributes mutator calls ApplyAnnotationsToPod with the OTLP-merged attrs
	// (no OA or metadata-enrichment mutator runs here — those require their namespace selectors to match).
	builder.Assess("metadata.dynatrace.com JSON annotation contains merged OTLP resource attributes", assessPodMetadataJSONAnnotation(sampleApp, expectedOTLP))
	builder.Assess("metadata.dynatrace.com/* individual annotations contain merged OTLP resource attributes", assessPodIndividualAnnotations(sampleApp, expectedOTLP))

	uninstallSampleApp(builder, sampleApp)

	return builder.Feature()
}

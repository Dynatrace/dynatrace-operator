//go:build e2e

package publicregistry

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/e2e/features/cloudnative"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/activegate"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8sdaemonset"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8snamespace"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8sstatefulset"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/registry"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/sample"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/tenant"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

// Feature verifies that public-registry images can be deployed by the operator using tag-based references.
// Covers: OneAgent DaemonSet, CodeModules, ActiveGate, EEC, KSPM, and OTelCollector.
func Feature(t *testing.T) features.Feature {
	return feature(t, "public-registry-images", "public-registry-sample", []dynakube.Option{
		dynakube.WithCustomOneAgentImage(registry.GetLatestOneAgentImageURI(t)),
		dynakube.WithCodeModulesImage(registry.GetLatestCodeModulesImageURI(t)),
		dynakube.WithCustomActiveGateImage(registry.GetLatestActiveGateImageURI(t)),
		dynakube.WithExtensionsEECImageRef(t),
		dynakube.WithKSPMImageRef(t),
		dynakube.WithOTelCollectorImageRef(t),
	})
}

// FeatureWithDigest is the same as Feature but uses digest-based image references ("repo@sha256:hash")
// to verify that the operator correctly handles pinned image digests across all components.
func FeatureWithDigest(t *testing.T) features.Feature {
	return feature(t, "public-registry-images-digest", "public-registry-digest-sample", []dynakube.Option{
		dynakube.WithCustomOneAgentImage(registry.GetLatestOneAgentImageDigestURI(t)),
		dynakube.WithCodeModulesImage(registry.GetLatestCodeModulesImageDigestURI(t)),
		dynakube.WithCustomActiveGateImage(registry.GetLatestActiveGateImageDigestURI(t)),
		dynakube.WithExtensionsEECImageRefDigest(t),
		dynakube.WithKSPMImageRefDigest(t),
		dynakube.WithOTelCollectorImageRefDigest(t),
	})
}

func feature(t *testing.T, featureName, sampleNS string, imageOpts []dynakube.Option) features.Feature {
	builder := features.New(featureName)
	secretConfig := tenant.GetSingleTenantSecret(t)

	const baseOptionCount = 6
	options := make([]dynakube.Option, 0, baseOptionCount+len(imageOpts))
	options = append(options,
		dynakube.WithAPIURL(secretConfig.APIURL),
		dynakube.WithCloudNativeSpec(cloudnative.DefaultCloudNativeSpec()),
		dynakube.WithActiveGate(),
		dynakube.WithExtensionsPrometheusEnabledSpec(true),
		dynakube.WithKSPM(),
		dynakube.WithTelemetryIngestEnabled(true),
	)
	options = append(options, imageOpts...)
	testDynakube := *dynakube.New(options...)

	sampleNamespace := *k8snamespace.New(sampleNS)
	sampleApp := sample.NewApp(t, &testDynakube, sample.WithNamespace(sampleNamespace), sample.AsDeployment())

	builder.Assess("create sample namespace", sampleApp.InstallNamespace())

	dynakube.Install(builder, &secretConfig, testDynakube)

	builder.Assess("install sample app", sampleApp.Install())
	cloudnative.AssessSampleInitContainers(builder, sampleApp)

	builder.Assess("ActiveGate started", k8sstatefulset.IsReady(activegate.GetActiveGateStateFulSetName(&testDynakube, "activegate"), testDynakube.Namespace))
	builder.Assess("EEC started", k8sstatefulset.IsReady(testDynakube.Extensions().GetExecutionControllerStatefulsetName(), testDynakube.Namespace))
	builder.Assess("kspm node config collector started", k8sdaemonset.IsReady(testDynakube.KSPM().GetDaemonSetName(), testDynakube.Namespace))
	builder.Assess("otel collector started", k8sstatefulset.IsReady(testDynakube.OtelCollectorStatefulsetName(), testDynakube.Namespace))

	builder.Teardown(sampleApp.Uninstall())

	return builder.Feature()
}

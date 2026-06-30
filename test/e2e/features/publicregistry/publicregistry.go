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

type componentImages struct {
	oneAgent    string
	activeGate  string
	codeModules string
	eec         string
	kspm        string
	otel        string
}

// Feature verifies that public-registry images can be deployed by the operator using tag-based references.
// Covers: OneAgent DaemonSet, CodeModules, ActiveGate, EEC, KSPM, and OTelCollector.
func Feature(t *testing.T) features.Feature {
	images := componentImages{
		oneAgent:    registry.GetLatestOneAgentImageURI(t),
		activeGate:  registry.GetLatestActiveGateImageURI(t),
		codeModules: registry.GetLatestCodeModulesImageURI(t),
		eec:         dynakube.GetLatestEECImageTagURI(t),
		kspm:        dynakube.GetLatestKSPMImageTagURI(t),
		otel:        dynakube.GetLatestOTelCollectorImageTagURI(t),
	}

	return feature(t, "public-registry-images", "public-registry-sample", []dynakube.Option{
		dynakube.WithCustomOneAgentImage(images.oneAgent),
		dynakube.WithCodeModulesImage(images.codeModules),
		dynakube.WithCustomActiveGateImage(images.activeGate),
		dynakube.WithExtensionsEECImageRef(t, images.eec),
		dynakube.WithKSPMImageRef(t, images.kspm),
		dynakube.WithOTelCollectorImageRef(t, images.otel),
	}, images)
}

// FeatureWithDigest is the same as Feature but uses digest-based image references ("repo@sha256:hash")
// to verify that the operator correctly handles pinned image digests across all components.
func FeatureWithDigest(t *testing.T) features.Feature {
	images := componentImages{
		oneAgent:    registry.GetLatestOneAgentImageDigestURI(t),
		activeGate:  registry.GetLatestActiveGateImageDigestURI(t),
		codeModules: registry.GetLatestCodeModulesImageDigestURI(t),
		eec:         dynakube.GetLatestEECImageDigestURI(t),
		kspm:        dynakube.GetLatestKSPMImageDigestURI(t),
		otel:        dynakube.GetLatestOTelCollectorImageDigestURI(t),
	}

	return feature(t, "public-registry-images-digest", "public-registry-digest-sample", []dynakube.Option{
		dynakube.WithCustomOneAgentImage(images.oneAgent),
		dynakube.WithCodeModulesImage(images.codeModules),
		dynakube.WithCustomActiveGateImage(images.activeGate),
		dynakube.WithExtensionsEECImageRef(t, images.eec),
		dynakube.WithKSPMImageRef(t, images.kspm),
		dynakube.WithOTelCollectorImageRef(t, images.otel),
	}, images)
}

func feature(t *testing.T, featureName, sampleNS string, imageOpts []dynakube.Option, images componentImages) features.Feature {
	builder := features.New(featureName)
	secretConfig := tenant.GetSingleTenantSecret(t)

	options := append([]dynakube.Option{
		dynakube.WithAPIURL(secretConfig.APIURL),
		dynakube.WithCloudNativeSpec(cloudnative.DefaultCloudNativeSpec()),
		dynakube.WithActiveGate(),
		dynakube.WithExtensionsPrometheusEnabledSpec(true),
		dynakube.WithKSPM(),
		dynakube.WithTelemetryIngestEnabled(true),
	}, imageOpts...,
	)

	testDynakube := *dynakube.New(options...)

	sampleNamespace := *k8snamespace.New(sampleNS)
	sampleApp := sample.NewApp(t, &testDynakube, sample.WithNamespace(sampleNamespace), sample.AsDeployment())

	builder.Assess("create sample namespace", sampleApp.InstallNamespace())

	dynakube.Install(builder, &secretConfig, testDynakube)

	builder.Assess("install sample app", sampleApp.Install())
	cloudnative.AssessSampleInitContainers(builder, sampleApp)

	agStatefulSetName := activegate.GetActiveGateStateFulSetName(&testDynakube, "activegate")
	builder.Assess("ActiveGate started", k8sstatefulset.IsReady(agStatefulSetName, testDynakube.Namespace))
	builder.Assess("EEC started", k8sstatefulset.IsReady(testDynakube.Extensions().GetExecutionControllerStatefulsetName(), testDynakube.Namespace))
	builder.Assess("kspm node config collector started", k8sdaemonset.IsReady(testDynakube.KSPM().GetDaemonSetName(), testDynakube.Namespace))
	builder.Assess("otel collector started", k8sstatefulset.IsReady(testDynakube.OtelCollectorStatefulsetName(), testDynakube.Namespace))

	builder.Assess("OneAgent DaemonSet uses expected image",
		k8sdaemonset.VerifyUsesImage(testDynakube.OneAgent().GetDaemonsetName(), testDynakube.Namespace, images.oneAgent))
	builder.Assess("ActiveGate StatefulSet uses expected image",
		k8sstatefulset.VerifyUsesImage(agStatefulSetName, testDynakube.Namespace, images.activeGate))
	builder.Assess("EEC StatefulSet uses expected image",
		k8sstatefulset.VerifyUsesImage(testDynakube.Extensions().GetExecutionControllerStatefulsetName(), testDynakube.Namespace, images.eec))
	builder.Assess("KSPM DaemonSet uses expected image",
		k8sdaemonset.VerifyUsesImage(testDynakube.KSPM().GetDaemonSetName(), testDynakube.Namespace, images.kspm))
	builder.Assess("OTel Collector StatefulSet uses expected image",
		k8sstatefulset.VerifyUsesImage(testDynakube.OtelCollectorStatefulsetName(), testDynakube.Namespace, images.otel))

	builder.Teardown(sampleApp.Uninstall())

	return builder.Feature()
}

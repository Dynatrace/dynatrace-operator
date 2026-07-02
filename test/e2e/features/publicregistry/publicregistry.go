//go:build e2e

package publicregistry

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/extensions"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/features/cloudnative"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/activegate"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8sdaemonset"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8sdeployment"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8snamespace"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8sstatefulset"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/platform"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/registry"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/sample"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

type componentImages struct {
	oneAgent    string
	activeGate  string
	codeModules string
	eec         string
	kspm        string
	otel        string
	dbExecutor  string
}

// Feature verifies that public-registry images can be deployed by the operator using tag-based references.
// Covers: OneAgent DaemonSet, CodeModules, ActiveGate, EEC, KSPM, and OTelCollector.
func Feature(t *testing.T) features.Feature {
	images := componentImages{
		oneAgent:    registry.GetLatestOneAgentImageTagURI(t),
		activeGate:  registry.GetLatestActiveGateImageTagURI(t),
		codeModules: registry.GetLatestCodeModulesImageTagURI(t),
		eec:         dynakube.GetLatestEECImageTagURI(t),
		kspm:        dynakube.GetLatestKSPMImageTagURI(t),
		otel:        dynakube.GetLatestOTelCollectorImageTagURI(t),
		dbExecutor:  dynakube.GetLatestDBExecutorImageTagURI(t),
	}

	return feature(t, "public-registry-images", "public-registry-sample", []dynakube.Option{
		dynakube.WithCustomOneAgentImage(images.oneAgent),
		dynakube.WithCodeModulesImage(images.codeModules),
		dynakube.WithCustomActiveGateImage(images.activeGate),
		dynakube.WithExtensionsEECImageRef(t, images.eec),
		dynakube.WithKSPMImageRef(t, images.kspm),
		dynakube.WithOTelCollectorImageRef(t, images.otel),
		dynakube.WithExtensionsDBExecutorImageRef(t, images.dbExecutor),
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
		dbExecutor:  dynakube.GetLatestDBExecutorImageDigestURI(t),
	}

	return feature(t, "public-registry-images-digest", "public-registry-digest-sample", []dynakube.Option{
		dynakube.WithCustomOneAgentImage(images.oneAgent),
		dynakube.WithCodeModulesImage(images.codeModules),
		dynakube.WithCustomActiveGateImage(images.activeGate),
		dynakube.WithExtensionsEECImageRef(t, images.eec),
		dynakube.WithKSPMImageRef(t, images.kspm),
		dynakube.WithOTelCollectorImageRef(t, images.otel),
		dynakube.WithExtensionsDBExecutorImageRef(t, images.dbExecutor),
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
		dynakube.WithExtensionsDatabases(extensions.DatabaseSpec{ID: "mysql"}),
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
	builder.Assess("KSPM node config collector started", k8sdaemonset.IsReady(testDynakube.KSPM().GetDaemonSetName(), testDynakube.Namespace))
	builder.Assess("OTelCollector started", k8sstatefulset.IsReady(testDynakube.OtelCollectorStatefulsetName(), testDynakube.Namespace))
	dbExecutorDeployName := testDynakube.Extensions().GetDatabaseDatasourceName("mysql")
	builder.Assess("DB executor deployment started", k8sdeployment.IsReady(dbExecutorDeployName, testDynakube.Namespace))

	builder.Assess("OneAgent DaemonSet uses expected image",
		k8sdaemonset.VerifyUsesImage(testDynakube.OneAgent().GetDaemonsetName(), testDynakube.Namespace, images.oneAgent))
	builder.Assess("ActiveGate StatefulSet uses expected image",
		k8sstatefulset.VerifyUsesImage(agStatefulSetName, testDynakube.Namespace, images.activeGate))
	builder.Assess("EEC StatefulSet uses expected image",
		k8sstatefulset.VerifyUsesImage(testDynakube.Extensions().GetExecutionControllerStatefulsetName(), testDynakube.Namespace, images.eec))
	builder.Assess("KSPM DaemonSet uses expected image",
		k8sdaemonset.VerifyUsesImage(testDynakube.KSPM().GetDaemonSetName(), testDynakube.Namespace, images.kspm))
	builder.Assess("OTelCollector StatefulSet uses expected image",
		k8sstatefulset.VerifyUsesImage(testDynakube.OtelCollectorStatefulsetName(), testDynakube.Namespace, images.otel))
	builder.Assess("DB executor deployment uses expected image",
		k8sdeployment.VerifyUsesImage(dbExecutorDeployName, testDynakube.Namespace, images.dbExecutor))

	builder.Assess("CodeModules status reports expected image",
		func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
			require.NoError(t, envConfig.Client().Resources().Get(ctx, testDynakube.Name, testDynakube.Namespace, &testDynakube))
			assert.Equal(t, images.codeModules, testDynakube.Status.CodeModules.ImageID)

			return ctx
		})

	builder.Teardown(sampleApp.Uninstall())

	return builder.Feature()
}

// FeatureLogMonitoring verifies that a logmonitoring-only DynaKube deploys with
// the expected tag-based image reference.
func FeatureLogMonitoring(t *testing.T) features.Feature {
	imageURI := dynakube.GetLatestLogMonitoringImageTagURI(t)

	return featureLogMonitoring(t, "public-registry-images-logmonitoring", imageURI)
}

// FeatureLogMonitoringWithDigest is the same as FeatureLogMonitoring but uses a
// digest-based image reference ("repo@sha256:hash").
func FeatureLogMonitoringWithDigest(t *testing.T) features.Feature {
	imageURI := dynakube.GetLatestLogMonitoringImageDigestURI(t)

	return featureLogMonitoring(t, "public-registry-images-digest-logmonitoring", imageURI)
}

func featureLogMonitoring(t *testing.T, featureName, imageURI string) features.Feature {
	builder := features.New(featureName)
	secretConfig := tenant.GetSingleTenantSecret(t)

	options := []dynakube.Option{
		dynakube.WithAPIURL(secretConfig.APIURL),
		dynakube.WithLogMonitoring(),
		dynakube.WithLogMonitoringImageRef(t, imageURI),
	}

	isOpenshift, err := platform.NewResolver().IsOpenshift()
	require.NoError(t, err)

	if isOpenshift {
		options = append(options, dynakube.WithAnnotations(map[string]string{
			exp.OAPrivilegedKey: "true",
		}))
	}

	testDynakube := *dynakube.New(options...)

	dynakube.Install(builder, &secretConfig, testDynakube)

	builder.Assess("LogMonitoring DaemonSet started", k8sdaemonset.IsReady(testDynakube.LogMonitoring().GetDaemonSetName(), testDynakube.Namespace))
	builder.Assess("LogMonitoring DaemonSet uses expected image",
		k8sdaemonset.VerifyUsesImage(testDynakube.LogMonitoring().GetDaemonSetName(), testDynakube.Namespace, imageURI))

	return builder.Feature()
}

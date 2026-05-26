//go:build e2e

package resourceattributes

import (
	"context"
	"encoding/json"
	"net/url"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	dkmetadata "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/otlp"
	activegateconsts "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	maputil "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/attributes"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator/otlp/resourceattributes"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/features/cloudnative"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/features/consts"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/features/usepublicregistry"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/activegate"
	dynakubeComponents "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8sdaemonset"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8sdeployment"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8snamespace"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/sample"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const sampleAppName = "static-ra-app"

var (
	globalAttrs = map[string]string{
		"deployment.environment": "global-env",
		"service.namespace":      "global-ns",
		"global.only.key":        "global-only-value",
	}
	oneAgentAdditional = map[string]string{
		"deployment.environment": "oneagent-env", // overrides global
		"oa.only.key":            "oa-only-value",
	}
	otlpAdditional = map[string]string{
		"deployment.environment": "otlp-env", // overrides global
		"otlp.only.key":          "otlp-only-value",
	}

	expectedOneAgent = map[string]string{
		"deployment.environment": "oneagent-env",
		"service.namespace":      "global-ns",
		"global.only.key":        "global-only-value",
		"oa.only.key":            "oa-only-value",
	}
	expectedOTLP = map[string]string{
		"deployment.environment": "otlp-env",
		"service.namespace":      "global-ns",
		"global.only.key":        "global-only-value",
		"otlp.only.key":          "otlp-only-value",
	}
)

func newSampleApp(t *testing.T, dk *dynakube.DynaKube, ns string, labels map[string]string) *sample.App {
	t.Helper()

	return sample.NewApp(t, dk,
		sample.WithName(sampleAppName),
		sample.WithNamespace(*k8snamespace.New(ns)),
		sample.AsDeployment(),
		sample.WithNamespaceLabels(labels),
		sample.WithImagePullSecret(consts.DevRegistryPullSecretName), // to be removed before merge
	)
}

func installSampleApp(b *features.FeatureBuilder, app *sample.App) {
	b.Assess("create sample namespace", app.InstallNamespace())
	// to be removed before merge
	b.Assess("create registry pull secret in sample namespace",
		usepublicregistry.CopyDevRegistrySecret(app.Namespace()))
	b.Assess("installing sample app", app.Install())
}

func uninstallSampleApp(b *features.FeatureBuilder, app *sample.App) {
	b.WithTeardown("uninstalling sample app", app.Uninstall())
	b.WithTeardown("deleting sample app namespace", k8snamespace.Delete(app.Namespace()))
}

func Combined(t *testing.T) features.Feature {
	builder := features.New("resource-attributes-combined")
	secretConfig := tenant.GetSingleTenantSecret(t)
	ns := "resource-attributes-combined"

	// expectedOTLPInAll is the effective OTEL_RESOURCE_ATTRIBUTES content
	// when OA and OTLP are both injected into the same pod.
	//
	// The metadata mutator runs first and writes individual metadata.dynatrace.com/<key>
	// annotations to the pod on a set-if-not-present basis. When the OTLP resource-attributes
	// mutator then calls NewPodAttributes, readPodAnnotationAttributes picks those up as
	// podAnnotations, which have higher combine-precedence than the OTLP dynakube attrs.
	// For keys present in both OA and OTLP additional attrs, OA therefore wins.
	expectedOTLPInAll := map[string]string{
		"deployment.environment": "oneagent-env", // OA annotation wins over OTLP dynakube attr
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
		// to be removed before merge
		dynakubeComponents.WithAnnotations(map[string]string{"feature.dynatrace.com/use-public-registry": "true"}),
		dynakubeComponents.WithCustomPullSecret(consts.DevRegistryPullSecretName),
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

	builder.Assess("OneAgent dt_node_metadata.properties contains merged OneAgent resource attributes", assessDTNodeMetadataProperties(testDynakube, expectedOneAgent))
	builder.Assess("ActiveGate deployment.properties ConfigMap contains global resource attributes", assessActiveGateDeploymentProperties(testDynakube, globalAttrs))

	installSampleApp(builder, sampleApp)

	builder.Assess("initcontainer contains args with additionalAttributes", assessInitContainerArgs(sampleApp, expectedOneAgent))
	builder.Assess("dt_metadata.json and dt_metadata.properties contains merged OneAgent resource attributes", assessDTMetadataFiles(sampleApp, expectedOneAgent))
	builder.Assess("OTEL_RESOURCE_ATTRIBUTES contains merged OTLP resource attributes (OA wins shared keys)", assessOTLPInjectionAttributes(sampleApp, expectedOTLPInAll))
	builder.Assess("metadata.dynatrace.com JSON annotation contains merged OneAgent resource attributes", assessPodMetadataJSONAnnotation(sampleApp, expectedOneAgent))
	builder.Assess("metadata.dynatrace.com/* individual annotations contain merged OneAgent resource attributes", assessPodIndividualAnnotations(sampleApp, expectedOneAgent))

	uninstallSampleApp(builder, sampleApp)

	return builder.Feature()
}

func assessInitContainerArgs(app *sample.App, expected map[string]string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resource := envConfig.Client().Resources()
		initContainer := app.GetInitContainer(ctx, t, resource, sample.InitContainerName)

		for k, v := range expected {
			assert.Containsf(t, initContainer.Args, attributes.ToArg(k, v),
				"init container %q args missing %s=%s", sample.InitContainerName, k, v)
		}

		return ctx
	}
}

func assessDTMetadataFiles(app *sample.App, expected map[string]string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resource := envConfig.Client().Resources()
		pod := app.GetPod(ctx, t, resource)

		properties := metadataenrichment.GetMetadataPropertiesFromPod(ctx, t, resource, pod)
		rawMetadata := metadataenrichment.GetRawMetadataFromPod(ctx, t, resource, pod)
		metadata := map[string]string{}
		require.NoError(t, json.Unmarshal(rawMetadata, &metadata))

		for k, v := range expected {
			assert.Equalf(t, v, properties[k], "dt_metadata.properties key %q in pod %s", k, pod.Name)
			assert.Equalf(t, v, metadata[k], "dt_metadata.json key %q in pod %s", k, pod.Name)
		}

		return ctx
	}
}

func assessDTNodeMetadataProperties(dk dynakube.DynaKube, expected map[string]string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		r := envConfig.Client().Resources()
		q := k8sdaemonset.NewQuery(ctx, r, client.ObjectKey{
			Name:      dk.OneAgent().GetDaemonsetName(),
			Namespace: dk.Namespace,
		})

		err := q.ForEachPod(func(pod corev1.Pod) {
			properties := metadataenrichment.GetNodeMetadataPropertiesFromPod(ctx, t, r, pod)
			for k, v := range expected {
				assert.Equalf(t, v, properties[k], "dt_metadata.properties key %q in pod %s", k, pod.Name)
			}
		})
		require.NoError(t, err)

		return ctx
	}
}

func assessOTLPInjectionAttributes(app *sample.App, expected map[string]string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resource := envConfig.Client().Resources()
		query := k8sdeployment.NewQuery(ctx, resource, client.ObjectKey{Name: app.Name(), Namespace: app.Namespace()})

		err := query.ForEachPod(func(p corev1.Pod) {
			require.NotEmptyf(t, p.Spec.Containers, "pod %s has no containers", p.Name)
			gotAttrs, ok := resourceattributes.NewAttributesFromEnv(p.Spec.Containers[0].Env, resourceattributes.OTELResourceAttributesEnv)
			require.Truef(t, ok, "OTEL_RESOURCE_ATTRIBUTES missing on pod %s", p.Name)

			for k, v := range expected {
				assert.Equalf(t, url.QueryEscape(v), gotAttrs[k], "OTEL_RESOURCE_ATTRIBUTES key %q in pod %s", k, p.Name)
			}
		})

		require.NoError(t, err)

		return ctx
	}
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

// assessPodMetadataJSONAnnotation checks that the pod's metadata.dynatrace.com JSON annotation
// contains all expected key-value pairs.
// This annotation is written by ApplyAnnotationsToPod (combineForJSONAnnotation case) and uses
// dynakube + namespaceAnnotations + rules + rulesPropagate + podAnnotations sources.
// When both OA and OTLP mutators are active the annotation is written once (SetAnnotationIfNotExists),
// so the first mutator's attributes win.
func assessPodMetadataJSONAnnotation(app *sample.App, expected map[string]string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resource := envConfig.Client().Resources()
		pod := app.GetPod(ctx, t, resource)

		jsonVal, ok := pod.Annotations[dkmetadata.Annotation]
		require.Truef(t, ok, "pod %s missing annotation %q", pod.Name, dkmetadata.Annotation)

		var parsed map[string]string
		require.NoError(t, json.Unmarshal([]byte(jsonVal), &parsed))

		for k, v := range expected {
			assert.Equalf(t, v, parsed[k], "JSON annotation key %q in pod %s", k, pod.Name)
		}

		return ctx
	}
}

// assessPodIndividualAnnotations checks that the pod's individual metadata.dynatrace.com/<key>
// annotations contain all expected key-value pairs.
// These annotations are written by ApplyAnnotationsToPod (combineForMetadataAnnotations case) and
// use workloadInfo + dynakube + namespaceAnnotations + rulesPropagate sources.
// SetAnnotationIfNotExists means pre-existing values are never overwritten.
func assessPodIndividualAnnotations(app *sample.App, expected map[string]string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resource := envConfig.Client().Resources()
		pod := app.GetPod(ctx, t, resource)

		for k, v := range expected {
			annotationKey := dkmetadata.Prefix + k
			got, ok := pod.Annotations[annotationKey]
			if assert.Truef(t, ok, "pod %s missing annotation %q", pod.Name, annotationKey) {
				assert.Equalf(t, v, got, "annotation %q in pod %s", annotationKey, pod.Name)
			}
		}

		return ctx
	}
}

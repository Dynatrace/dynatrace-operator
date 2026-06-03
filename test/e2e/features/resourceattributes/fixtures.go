//go:build e2e

package resourceattributes

import (
	"context"
	"encoding/json"
	"maps"
	"net/url"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	dkmetadata "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/attributes"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator/otlp/resourceattributes"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/features/consts"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/features/usepublicregistry"
	dynakubeComponents "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8sdaemonset"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8sdeployment"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8snamespace"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/sample"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

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
	// exclusiveAttrs are resource attributes that are configured for exactly one
	// injection path. They must only ever show up on that path and must never
	// leak into the other one (e.g. an OTLP-only attribute appearing in the
	// OneAgent dt_metadata, or a OneAgent-only attribute appearing in
	// OTEL_RESOURCE_ATTRIBUTES).
	exclusiveAttrs = map[string]string{
		"oa.only.key":   "oa-only-value",
		"otlp.only.key": "otlp-only-value",
	}
)

const sampleAppName = "static-ra-app"

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

// to be removed before merge
func devRegistryOptions() dynakubeComponents.Option {
	return func(dk *dynakube.DynaKube) {
		dynakubeComponents.WithAnnotations(map[string]string{"feature.dynatrace.com/use-public-registry": "true"})(dk)
		dynakubeComponents.WithCustomPullSecret(consts.DevRegistryPullSecretName)(dk)
	}
}

func assessInitContainerArgs(app *sample.App, expected map[string]string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resource := envConfig.Client().Resources()
		initContainer := app.GetInitContainer(ctx, t, resource, sample.InitContainerName)

		for k, v := range expected {
			assert.Containsf(t, initContainer.Args, attributes.ToArg(k, v),
				"init container %q args missing %s=%s", sample.InitContainerName, k, v)
		}

		for k, v := range forbiddenAttrs(expected) {
			assert.NotContainsf(t, initContainer.Args, attributes.ToArg(k, v),
				"init container %q args leaked %s=%s", sample.InitContainerName, k, v)
		}

		return ctx
	}
}

func assessDTMetadataFiles(dk dynakube.DynaKube, app *sample.App, expected map[string]string) features.Func {
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

		expectedDefaults := buildExpectedMetadataEnrichmentDefaults(ctx, t, envConfig, dk, app)
		for k, v := range expectedDefaults {
			assert.Equalf(t, v, properties[k], "dt_metadata.properties key %q in pod %s", k, pod.Name)
			assert.Equalf(t, v, metadata[k], "dt_metadata.json key %q in pod %s", k, pod.Name)
		}

		for k := range forbiddenAttrs(expected) {
			assert.NotContainsf(t, properties, k, "dt_metadata.properties leaked key %q in pod %s", k, pod.Name)
			assert.NotContainsf(t, metadata, k, "dt_metadata.json leaked key %q in pod %s", k, pod.Name)
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

		expectedDefaults := buildExpectedNodeDefaults(ctx, t, envConfig, dk)
		forbidden := forbiddenAttrs(expected)
		err := q.ForEachPod(func(pod corev1.Pod) {
			properties := metadataenrichment.GetNodeMetadataPropertiesFromPod(ctx, t, r, pod)
			for k, v := range expected {
				assert.Equalf(t, v, properties[k], "dt_node_metadata.properties key %q in pod %s", k, pod.Name)
			}
			for k, v := range expectedDefaults {
				assert.Equalf(t, v, properties[k], "dt_node_metadata.properties key %q in pod %s", k, pod.Name)
			}
			for k := range forbidden {
				assert.NotContainsf(t, properties, k, "dt_node_metadata.properties leaked key %q in pod %s", k, pod.Name)
			}
		})
		require.NoError(t, err)

		return ctx
	}
}

func assessOTLPInjectionAttributes(dk dynakube.DynaKube, app *sample.App, expected map[string]string) features.Func {
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

			expectedDefaults := buildExpectedOTLPDefaults(ctx, t, envConfig, dk, app)
			for k, v := range expectedDefaults {
				assert.Equalf(t, v, gotAttrs[k], "OTEL_RESOURCE_ATTRIBUTES key %q in pod %s", k, p.Name)
			}

			for k := range forbiddenAttrs(expected) {
				assert.NotContainsf(t, gotAttrs, k, "OTEL_RESOURCE_ATTRIBUTES leaked key %q in pod %s", k, p.Name)
			}
		})

		require.NoError(t, err)

		return ctx
	}
}

func assessOTLPInjectionAttributesAbsent(app *sample.App) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resource := envConfig.Client().Resources()
		query := k8sdeployment.NewQuery(ctx, resource, client.ObjectKey{Name: app.Name(), Namespace: app.Namespace()})

		err := query.ForEachPod(func(p corev1.Pod) {
			require.NotEmptyf(t, p.Spec.Containers, "pod %s has no containers", p.Name)
			_, ok := resourceattributes.NewAttributesFromEnv(p.Spec.Containers[0].Env, resourceattributes.OTELResourceAttributesEnv)
			assert.Falsef(t, ok, "%s must be absent on pod %s when OTLP is not configured", resourceattributes.OTELResourceAttributesEnv, p.Name)
		})
		require.NoError(t, err)

		return ctx
	}
}

// assessPodMetadataJSONAnnotation checks that the pod's metadata.dynatrace.com JSON annotation
// contains all expected key-value pairs and workload info attributes (k8s.workload.kind/name).
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

		assert.Equalf(t, app.Kind(), parsed["k8s.workload.kind"], "JSON blob k8s.workload.kind in pod %s", pod.Name)
		assert.Equalf(t, app.Name(), parsed["k8s.workload.name"], "JSON blob k8s.workload.name in pod %s", pod.Name)

		assert.Equalf(t, app.Kind(), pod.Annotations[dkmetadata.Annotation+"/"+"k8s.workload.kind"], "individual annotation k8s.workload.kind in pod %s", pod.Name)
		assert.Equalf(t, app.Name(), pod.Annotations[dkmetadata.Annotation+"/"+"k8s.workload.name"], "individual annotation k8s.workload.name in pod %s", pod.Name)

		return ctx
	}
}

// assessDynakubeAttrsNotInIndividualAnnotations verifies that DynaKube resource attributes
// are NOT written as individual metadata.dynatrace.com/<key> annotations.
func assessDynakubeAttrsNotInIndividualAnnotations(app *sample.App, dynakubeAttrs map[string]string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resource := envConfig.Client().Resources()
		pod := app.GetPod(ctx, t, resource)

		for k := range dynakubeAttrs {
			annotationKey := dkmetadata.Prefix + k
			assert.NotContainsf(t, pod.Annotations, annotationKey, "DynaKube resource attr %q must not appear as individual annotation in pod %s", annotationKey, pod.Name)
		}

		return ctx
	}
}

// forbiddenAttrs returns the exclusive resource attributes that must not be
// present given the expected attributes: every exclusive attribute that is not
// part of expected would be a leak from the other injection path.
func forbiddenAttrs(expected map[string]string) map[string]string {
	forbidden := map[string]string{}
	for k, v := range exclusiveAttrs {
		if _, ok := expected[k]; !ok {
			forbidden[k] = v
		}
	}

	return forbidden
}

func buildExpectedOTLPDefaults(ctx context.Context, t *testing.T, envConfig *envconf.Config, dk dynakube.DynaKube, app *sample.App) map[string]string {
	expectedDefaults := make(map[string]string)
	maps.Copy(expectedDefaults, buildExpectedDefaults(ctx, t, envConfig, dk, app))
	maps.Copy(expectedDefaults, buildExpectedPodDefaultsOTLP())

	return expectedDefaults
}

func buildExpectedMetadataEnrichmentDefaults(ctx context.Context, t *testing.T, envConfig *envconf.Config, dk dynakube.DynaKube, app *sample.App) map[string]string {
	expectedDefaults := make(map[string]string)
	maps.Copy(expectedDefaults, buildExpectedDefaults(ctx, t, envConfig, dk, app))
	maps.Copy(expectedDefaults, buildExpectedPodDefaultsMetadataEnrichment(ctx, t, envConfig, app))

	return expectedDefaults
}

func buildExpectedDefaults(ctx context.Context, t *testing.T, envConfig *envconf.Config, dk dynakube.DynaKube, app *sample.App) map[string]string {
	expectedDefaults := make(map[string]string)

	err := envConfig.Client().Resources().Get(ctx, dk.Name, dk.Namespace, &dk)
	require.NoError(t, err)

	expectedDefaults["k8s.workload.kind"] = app.Kind()
	expectedDefaults["k8s.workload.name"] = app.Name()
	expectedDefaults["k8s.namespace.name"] = app.Namespace()
	expectedDefaults["k8s.cluster.uid"] = dk.Status.KubeSystemUUID
	expectedDefaults["k8s.cluster.name"] = dk.Status.KubernetesClusterName
	expectedDefaults["dt.entity.kubernetes_cluster"] = dk.Status.KubernetesClusterMEID
	expectedDefaults["k8s.container.name"] = app.ContainerName()

	return expectedDefaults
}

func buildExpectedNodeDefaults(ctx context.Context, t *testing.T, envConfig *envconf.Config, dk dynakube.DynaKube) map[string]string {
	err := envConfig.Client().Resources().Get(ctx, dk.Name, dk.Namespace, &dk)
	require.NoError(t, err)

	return map[string]string{
		"k8s.cluster.uid":              dk.Status.KubeSystemUUID,
		"k8s.cluster.name":             dk.Status.KubernetesClusterName,
		"dt.entity.kubernetes_cluster": dk.Status.KubernetesClusterMEID,
	}
}

func buildExpectedPodDefaultsOTLP() map[string]string {
	expectedDefaults := make(map[string]string)
	expectedDefaults["k8s.pod.name"] = "$(K8S_PODNAME)"
	expectedDefaults["k8s.pod.uid"] = "$(K8S_PODUID)"
	expectedDefaults["k8s.node.name"] = "$(K8S_NODE_NAME)"

	return expectedDefaults
}

func buildExpectedPodDefaultsMetadataEnrichment(ctx context.Context, t *testing.T, envConfig *envconf.Config, app *sample.App) map[string]string {
	pod := app.GetPod(ctx, t, envConfig.Client().Resources())

	expectedDefaults := make(map[string]string)
	expectedDefaults["k8s.pod.name"] = pod.Name
	expectedDefaults["k8s.pod.uid"] = string(pod.UID)
	expectedDefaults["k8s.node.name"] = pod.Spec.NodeName

	return expectedDefaults
}

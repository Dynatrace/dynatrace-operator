//go:build e2e

package applicationmonitoring

import (
	"context"
	"fmt"
	"net/url"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/otlp"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator/otlp/exporter"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator/otlp/resourceattributes"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	dynakubeComponents "github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/deployment"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sample"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

// Verification of the OTLP exporter configuration injection.
// The test creates a Dynakube with OTLP exporter configuration specifying a namespace selector
// and then deploys sample applications in namespaces that either match or do not match the selector.
// It verifies that the OTLP exporter environment variables are injected only into the pods
// of applications running in namespaces that match the selector.
func OTLPExporterConfiguration(t *testing.T) features.Feature {
	builder := features.New("otlp-exporter-configuration")
	secretConfig := tenant.GetSingleTenantSecret(t)

	testDynakube := *dynakubeComponents.New(
		dynakubeComponents.WithAPIURL(secretConfig.APIURL),
		dynakubeComponents.WithApplicationMonitoringSpec(&oneagent.ApplicationMonitoringSpec{}),
	)
	// configure OTLP exporter signals + namespace selector
	testDynakube.Spec.OTLPExporterConfiguration = &otlp.ExporterConfigurationSpec{
		NamespaceSelector: metav1.LabelSelector{MatchLabels: map[string]string{"otlp-inject": testDynakube.Name}},
		Signals: otlp.SignalConfiguration{
			Metrics: &otlp.MetricsSignal{},
			Logs:    &otlp.LogsSignal{},
			Traces:  &otlp.TracesSignal{},
		},
	}

	metadataAnnotations := map[string]string{
		"metadata.dynatrace.com/service.name": "checkout service",
		"metadata.dynatrace.com/custom.key":   "value:with/special chars",
	}

	type testCase struct {
		name                 string
		app                  *sample.App
		assess               func(sampleApp *sample.App, expectedBase string) features.Func
		expectedBaseEndpoint string
	}

	matchingLabels := testDynakube.Spec.OTLPExporterConfiguration.NamespaceSelector.MatchLabels
	nonMatchingLabels := map[string]string{"other-label": "no-match"}
	baseEndpoint := secretConfig.APIURL + "/v2/otlp"

	testCases := []testCase{
		{
			name: "deployment matching namespace selector",
			app: sample.NewApp(t, &testDynakube,
				sample.WithName("deploy-otlp"),
				sample.AsDeployment(),
				sample.WithNamespaceLabels(matchingLabels),
				sample.WithAnnotations(metadataAnnotations),
			),
			assess:               deploymentPodsHaveOTLPExporterEnvVarsInjected,
			expectedBaseEndpoint: baseEndpoint,
		},
		{
			name: "pod matching namespace selector",
			app: sample.NewApp(t, &testDynakube,
				sample.WithName("pod-otlp"),
				sample.WithNamespaceLabels(matchingLabels),
				sample.WithAnnotations(metadataAnnotations),
			),
			assess:               podHasOTLPExporterEnvVarsInjected,
			expectedBaseEndpoint: baseEndpoint,
		},
		{
			name: "deployment non-matching namespace selector",
			app: sample.NewApp(t, &testDynakube,
				sample.WithName("deploy-no-otlp"),
				sample.AsDeployment(),
				sample.WithNamespaceLabels(nonMatchingLabels),
			),
			assess:               deploymentPodsHaveNoOTLPExporterEnvVarsInjected,
			expectedBaseEndpoint: baseEndpoint,
		},
		{
			name: "pod non-matching namespace selector",
			app: sample.NewApp(t, &testDynakube,
				sample.WithName("pod-no-otlp"),
				sample.WithNamespaceLabels(nonMatchingLabels),
			),
			assess:               podHasNoOTLPExporterEnvVarsInjected,
			expectedBaseEndpoint: baseEndpoint,
		},
	}

	dynakubeComponents.Install(builder, helpers.LevelAssess, &secretConfig, testDynakube)
	for _, tc := range testCases {
		builder.Assess(fmt.Sprintf("%s: Installing sample app", tc.name), tc.app.Install())
		builder.Assess(fmt.Sprintf("%s: Checking sample app", tc.name), tc.assess(tc.app, tc.expectedBaseEndpoint))
		builder.WithTeardown(fmt.Sprintf("%s: Uninstalling sample app", tc.name), tc.app.Uninstall())
	}
	dynakubeComponents.Delete(builder, helpers.LevelTeardown, testDynakube)

	return builder.Feature()
}

// Assessors
func podHasOTLPExporterEnvVarsInjected(app *sample.App, expectedBase string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		pods := app.GetPods(ctx, t, envConfig.Client().Resources())
		require.NotEmpty(t, pods.Items)
		assertOTLPEnvVarsPresentWithResourceAttributes(t, &pods.Items[0], expectedBase)

		return ctx
	}
}

func podHasNoOTLPExporterEnvVarsInjected(app *sample.App, _ string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		pods := app.GetPods(ctx, t, envConfig.Client().Resources())
		require.NotEmpty(t, pods.Items)
		assertOTLPEnvVarsAbsent(t, &pods.Items[0])

		return ctx
	}
}

func deploymentPodsHaveOTLPExporterEnvVarsInjected(app *sample.App, expectedBase string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		query := deployment.NewQuery(ctx, envConfig.Client().Resources(), client.ObjectKey{Name: app.Name(), Namespace: app.Namespace()})

		err := query.ForEachPod(func(p corev1.Pod) { assertOTLPEnvVarsPresentWithResourceAttributes(t, &p, expectedBase) })
		require.NoError(t, err)

		return ctx
	}
}

func deploymentPodsHaveNoOTLPExporterEnvVarsInjected(app *sample.App, _ string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		query := deployment.NewQuery(ctx, envConfig.Client().Resources(), client.ObjectKey{Name: app.Name(), Namespace: app.Namespace()})
		err := query.ForEachPod(func(p corev1.Pod) { assertOTLPEnvVarsAbsent(t, &p) })
		require.NoError(t, err)

		return ctx
	}
}

// Assertions
func assertOTLPEnvVarsPresent(t *testing.T, podItem *corev1.Pod, expectedBase string) {
	require.NotNil(t, podItem)
	require.NotEmpty(t, podItem.Spec.Containers)
	appContainer := podItem.Spec.Containers[0]
	envMap := map[string]corev1.EnvVar{}
	for _, e := range appContainer.Env {
		envMap[e.Name] = e
	}

	for name, suffix := range map[string]string{exporter.OTLPTraceEndpointEnv: "/v1/traces", exporter.OTLPLogsEndpointEnv: "/v1/logs", exporter.OTLPMetricsEndpointEnv: "/v1/metrics"} {
		envVar := env.FindEnvVar(appContainer.Env, name)
		assert.NotNil(t, envVar, "%s env var missing", name)
		if envVar != nil {
			assert.Equal(t, expectedBase+suffix, envVar.Value, "%s value", name)
		}
	}
	for _, name := range []string{exporter.OTLPTraceHeadersEnv, exporter.OTLPLogsHeadersEnv, exporter.OTLPMetricsHeadersEnv} {
		envVar := env.FindEnvVar(appContainer.Env, name)
		assert.NotNil(t, envVar, "%s env var missing", name)
		if envVar != nil {
			assert.Equal(t, exporter.OTLPAuthorizationHeader, envVar.Value, "%s header", name)
		}
	}
	temporalityPreferenceEnv := env.FindEnvVar(appContainer.Env, exporter.OTLPMetricsExporterTemporalityPreference)
	assert.NotNil(t, temporalityPreferenceEnv, "%s env var missing", exporter.OTLPMetricsExporterTemporalityPreference)
	if temporalityPreferenceEnv != nil {
		assert.Equal(t, exporter.OTLPMetricsExporterAggregationTemporalityDelta, temporalityPreferenceEnv.Value, "%s value", exporter.OTLPMetricsExporterTemporalityPreference)
	}
	tokenEnv := env.FindEnvVar(appContainer.Env, exporter.DynatraceAPITokenEnv)
	assert.NotNil(t, tokenEnv, "%s env var missing", exporter.DynatraceAPITokenEnv)
	if tokenEnv != nil {
		require.NotNil(t, tokenEnv.ValueFrom)
		require.NotNil(t, tokenEnv.ValueFrom.SecretKeyRef)
		assert.Equal(t, consts.OTLPExporterSecretName, tokenEnv.ValueFrom.SecretKeyRef.Name)
		assert.Equal(t, dynatrace.DataIngestToken, tokenEnv.ValueFrom.SecretKeyRef.Key)
	}
}

func assertOTLPEnvVarsAbsent(t *testing.T, podItem *corev1.Pod) {
	require.NotNil(t, podItem)
	require.NotEmpty(t, podItem.Spec.Containers)
	appContainer := podItem.Spec.Containers[0]
	for _, name := range []string{exporter.OTLPTraceEndpointEnv, exporter.OTLPLogsEndpointEnv, exporter.OTLPMetricsEndpointEnv, exporter.OTLPTraceHeadersEnv, exporter.OTLPLogsHeadersEnv, exporter.OTLPMetricsHeadersEnv, exporter.DynatraceAPITokenEnv, resourceattributes.OTELResourceAttributesEnv} {
		for _, e := range appContainer.Env {
			assert.NotEqual(t, name, e.Name, "%s should not be injected", name)
		}
	}
}

func assertOTLPEnvVarsPresentWithResourceAttributes(t *testing.T, podItem *corev1.Pod, expectedBase string) {
	assertOTLPEnvVarsPresent(t, podItem, expectedBase)
	gotResourceAttributes, ok := resourceattributes.NewAttributesFromEnv(podItem.Spec.Containers[0].Env, resourceattributes.OTELResourceAttributesEnv)

	require.True(t, ok, "OTEL_RESOURCE_ATTRIBUTES missing")
	assert.Equal(t, url.QueryEscape("checkout service"), gotResourceAttributes["service.name"])       // annotation encoded
	assert.Equal(t, url.QueryEscape("value:with/special chars"), gotResourceAttributes["custom.key"]) // annotation encoded
	assert.Equal(t, podItem.Namespace, gotResourceAttributes["k8s.namespace.name"])
}

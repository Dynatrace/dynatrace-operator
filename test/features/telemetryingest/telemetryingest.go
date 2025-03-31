//go:build e2e

package telemetryingest

import (
	"context"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube/activegate"
	otelcconsts "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/otelc/consts"
	"github.com/Dynatrace/dynatrace-operator/test/features/consts"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	componentActiveGate "github.com/Dynatrace/dynatrace-operator/test/helpers/components/activegate"
	componentDynakube "github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/secret"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/statefulset"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/logs"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tls"
	"github.com/Dynatrace/dynatrace-operator/test/project"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	activeGateComponent   = "activegate"
	TelemetryIngestTLSCrt = "custom-cas/tls-telemetry-ingest.crt"
	TelemetryIngestTLSKey = "custom-cas/tls-telemetry-ingest.key"
)

// Rollout of OTel collector when no ActiveGate is configured in the Dynakube
func WithPublicActiveGate(t *testing.T) features.Feature {
	builder := features.New("telemetryingest-with-public-ag-components-rollout")

	secretConfig := tenant.GetSingleTenantSecret(t)

	options := []componentDynakube.Option{
		componentDynakube.WithApiUrl(secretConfig.ApiUrl),
		componentDynakube.WithTelemetryIngestEnabled(true),
	}

	testDynakube := *componentDynakube.New(options...)

	componentDynakube.Install(builder, helpers.LevelAssess, &secretConfig, testDynakube)

	builder.Assess("otel collector started", statefulset.WaitFor(testDynakube.OtelCollectorStatefulsetName(), testDynakube.Namespace))
	builder.Assess("otel collector config created", checkOtelCollectorConfig(&testDynakube))
	builder.Assess("otel collector service created", checkOtelCollectorService(&testDynakube))

	componentDynakube.Delete(builder, helpers.LevelTeardown, testDynakube)

	builder.WithTeardown("deleted tenant secret", tenant.DeleteTenantSecret(testDynakube.Name, testDynakube.Namespace))

	return builder.Feature()
}

// Rollout of OTel collector and a local in-cluster ActiveGate. Make sure that components are cleaned up after telemetryIngest gets disabled.
func WithLocalActiveGateAndCleanup(t *testing.T) features.Feature {
	builder := features.New("telemetryingest-with-local-active-gate-component-rollout-and-cleanup-after-disable")

	secretConfig := tenant.GetSingleTenantSecret(t)

	optionsTelemetryIngestEnabled := []componentDynakube.Option{
		componentDynakube.WithApiUrl(secretConfig.ApiUrl),
		componentDynakube.WithTelemetryIngestEnabled(true, "zipkin"),
		componentDynakube.WithActiveGateModules(activegate.KubeMonCapability.DisplayName),
		componentDynakube.WithActiveGateTLSSecret(consts.AgSecretName),
	}

	testDynakube := *componentDynakube.New(optionsTelemetryIngestEnabled...)

	agSecret, err := createAgTlsSecret(testDynakube.Namespace)
	require.NoError(t, err, "failed to create ag-tls secret")
	builder.Assess("create AG TLS secret", secret.Create(agSecret))

	componentDynakube.Install(builder, helpers.LevelAssess, &secretConfig, testDynakube)
	builder.Assess("active gate pod is running", checkActiveGateContainer(&testDynakube))

	builder.Assess("otel collector started", statefulset.WaitFor(testDynakube.OtelCollectorStatefulsetName(), testDynakube.Namespace))
	builder.Assess("otel collector config created", checkOtelCollectorConfig(&testDynakube))
	builder.Assess("otel collector service created", checkOtelCollectorService(&testDynakube))
	builder.Assess("otel collector endpoint configmap created", checkOtelCollectorEndpointConfigMap(&testDynakube))

	optionsTelemetryIngestDisabled := []componentDynakube.Option{
		componentDynakube.WithApiUrl(secretConfig.ApiUrl),
		componentDynakube.WithTelemetryIngestEnabled(false),
		componentDynakube.WithActiveGateModules(activegate.KubeMonCapability.DisplayName),
		componentDynakube.WithActiveGateTLSSecret(consts.AgSecretName),
	}

	testDynakubeNoTelemetryIngest := *componentDynakube.New(optionsTelemetryIngestDisabled...)
	componentDynakube.Update(builder, helpers.LevelAssess, testDynakubeNoTelemetryIngest)

	builder.Assess("otel collector shutdown", waitForShutdown(testDynakubeNoTelemetryIngest.OtelCollectorStatefulsetName(), testDynakubeNoTelemetryIngest.Namespace))
	builder.Assess("otel collector config removed", checkOtelCollectorConfigRemoved(&testDynakubeNoTelemetryIngest))
	builder.Assess("otel collector service removed", checkOtelCollectorServiceRemoved(&testDynakubeNoTelemetryIngest))
	builder.Assess("otel collector endpoint configmap removed", checkOtelCollectorEndpointConfigMapRemoved(&testDynakubeNoTelemetryIngest))

	componentDynakube.Delete(builder, helpers.LevelTeardown, testDynakubeNoTelemetryIngest)

	builder.WithTeardown("deleted tenant secret", tenant.DeleteTenantSecret(testDynakubeNoTelemetryIngest.Name, testDynakubeNoTelemetryIngest.Namespace))

	return builder.Feature()
}

// Rollout of OTel collector with TLS secret to secure the telemetryIngest endpoints
func WithTelemetryIngestEndpointTLS(t *testing.T) features.Feature {
	builder := features.New("telemetryingest-with-otel-collector-endpoint-tls")

	secretConfig := tenant.GetSingleTenantSecret(t)

	options := []componentDynakube.Option{
		componentDynakube.WithApiUrl(secretConfig.ApiUrl),
		componentDynakube.WithTelemetryIngestEnabled(true),
		componentDynakube.WithTelemetryIngestEndpointTLS(consts.TelemetryIngestTLSSecretName),
	}

	testDynakube := *componentDynakube.New(options...)

	tlsSecret, err := tls.CreateTestdataTLSSecret(testDynakube.Namespace, consts.TelemetryIngestTLSSecretName, TelemetryIngestTLSKey, TelemetryIngestTLSCrt)
	require.NoError(t, err, "failed to create TLS secret for otel collector endpoints")

	builder.Assess("create OTel collector endpoint TLS secret", secret.Create(tlsSecret))

	componentDynakube.Install(builder, helpers.LevelAssess, &secretConfig, testDynakube)

	builder.Assess("otel collector started", statefulset.WaitFor(testDynakube.OtelCollectorStatefulsetName(), testDynakube.Namespace))
	builder.Assess("otel collector config created", checkOtelCollectorConfig(&testDynakube))
	builder.Assess("otel collector service created", checkOtelCollectorService(&testDynakube))
	builder.Assess("otel collector endpoint configmap created", checkOtelCollectorEndpointConfigMap(&testDynakube))

	componentDynakube.Delete(builder, helpers.LevelTeardown, testDynakube)
	builder.WithTeardown("deleted tenant secret", tenant.DeleteTenantSecret(testDynakube.Name, testDynakube.Namespace))
	builder.WithTeardown("deleted OTel collector endpoint TLS secret", secret.Delete(tlsSecret))

	return builder.Feature()
}

// Make sure the Otel collector configuration is updated and pods are restarted when protocols for telemetryIngest change
func OtelCollectorConfigUpdate(t *testing.T) features.Feature {
	builder := features.New("telemetryingest-configuration-update")

	secretConfig := tenant.GetSingleTenantSecret(t)

	optionsZipkin := []componentDynakube.Option{
		componentDynakube.WithApiUrl(secretConfig.ApiUrl),
		componentDynakube.WithTelemetryIngestEnabled(true, "zipkin"),
	}

	testDynakubeZipkin := *componentDynakube.New(optionsZipkin...)

	componentDynakube.Install(builder, helpers.LevelAssess, &secretConfig, testDynakubeZipkin)

	builder.Assess("otel collector started", statefulset.WaitFor(testDynakubeZipkin.OtelCollectorStatefulsetName(), testDynakubeZipkin.Namespace))
	builder.Assess("otel collector config created", checkOtelCollectorConfig(&testDynakubeZipkin))
	builder.Assess("otel collector service created", checkOtelCollectorService(&testDynakubeZipkin))
	builder.Assess("otel collector endpoint configmap created", checkOtelCollectorEndpointConfigMap(&testDynakubeZipkin))

	var zipkinConfigResourceVersion string
	builder.Assess("otel collector configuration timestamp", getOtelCollectorConfigResourceVersion(&testDynakubeZipkin, &zipkinConfigResourceVersion))

	var zipkinPodStartTs time.Time
	builder.Assess("otel collector pod creation timestamp", getOtelCollectorPodTimestamp(&testDynakubeZipkin, &zipkinPodStartTs))

	optionsJaeger := []componentDynakube.Option{
		componentDynakube.WithApiUrl(secretConfig.ApiUrl),
		componentDynakube.WithTelemetryIngestEnabled(true, "jaeger"),
	}

	testDynakubeJaeger := *componentDynakube.New(optionsJaeger...)
	componentDynakube.Update(builder, helpers.LevelAssess, testDynakubeJaeger)

	builder.Assess("otel collector started", statefulset.WaitFor(testDynakubeJaeger.OtelCollectorStatefulsetName(), testDynakubeJaeger.Namespace))
	builder.Assess("otel collector config created", checkOtelCollectorConfig(&testDynakubeJaeger))
	builder.Assess("otel collector service created", checkOtelCollectorService(&testDynakubeJaeger))

	var jaegerConfigResourceVersion string
	builder.Assess("otel collector configuration timestamp", getOtelCollectorConfigResourceVersion(&testDynakubeJaeger, &jaegerConfigResourceVersion))
	builder.Assess("otel collector configuration updated", func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
		assert.NotEqual(t, jaegerConfigResourceVersion, zipkinConfigResourceVersion)

		return ctx
	})

	var jaegerPodStartTs time.Time
	builder.Assess("otel collector pod creation timestamp", getOtelCollectorPodTimestamp(&testDynakubeJaeger, &jaegerPodStartTs))
	builder.Assess("otel collector pod restarted", func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
		assert.Greater(t, jaegerPodStartTs, zipkinPodStartTs)

		return ctx
	})

	componentDynakube.Delete(builder, helpers.LevelTeardown, testDynakubeJaeger)

	builder.WithTeardown("deleted tenant secret", tenant.DeleteTenantSecret(testDynakubeJaeger.Name, testDynakubeJaeger.Namespace))

	return builder.Feature()
}

func checkActiveGateContainer(dk *dynakube.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()

		var activeGatePod corev1.Pod
		require.NoError(t, resources.WithNamespace(dk.Namespace).Get(ctx, componentActiveGate.GetActiveGatePodName(dk, activeGateComponent), dk.Namespace, &activeGatePod))

		require.NotNil(t, activeGatePod.Spec)
		require.NotEmpty(t, activeGatePod.Spec.Containers)

		assertTelemetryIngestActiveGateModulesAreActive(ctx, t, envConfig, dk)

		return ctx
	}
}

func assertTelemetryIngestActiveGateModulesAreActive(ctx context.Context, t *testing.T, envConfig *envconf.Config, dk *dynakube.DynaKube) {
	var expectedModules = []string{"log_analytics_collector", "otlp_ingest"}
	var expectedServices = []string{"generic_ingest"}

	log := componentActiveGate.ReadActiveGateLog(ctx, t, envConfig, dk, activeGateComponent)

	/* componentActiveGate 2025-03-24 15:08:02 UTC INFO    [<exq67461>] [<collector.services>, ServicesManager] Services active: [generic_filecache, local_support_archive, generic_ingest] */
	servicesLog := logs.FindLineContainingText(log, "Services active:")
	for _, service := range expectedServices {
		assert.Contains(t, servicesLog, service, "ActiveGate services is not active: '"+service+"'")
	}

	head := strings.SplitAfter(log, "[<collector.modules>, ModulesManager] Modules:")
	require.Len(t, head, 2, "list of AG active modules not found")

	tail := strings.SplitAfter(head[1], "Lifecycle listeners:")
	require.Len(t, head, 2, "list of AG active modules not found")

	/*
		Expected log messages of the Gateway process:
			`Active:
				    log_analytics_collector"
				    generic_ingest"
				    otlp_ingest"
			Lifecycle listeners:`

		Warning: modules are printed in random order.
	*/
	for _, module := range expectedModules {
		assert.Contains(t, tail[0], module, "ActiveGate module is not active: '"+module+"'")
	}
}

func checkOtelCollectorConfig(dk *dynakube.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		otelCollectorConfig, err := getOtelCollectorConfigMap(dk, ctx, envConfig)
		require.NoError(t, err, "failed to get otel collector config")

		require.NotNil(t, otelCollectorConfig.Data)

		return ctx
	}
}

func checkOtelCollectorConfigRemoved(dk *dynakube.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		_, err := getOtelCollectorConfigMap(dk, ctx, envConfig)
		require.Error(t, err)
		assert.True(t, k8serrors.IsNotFound(err), "ConfigMap still exists")

		return ctx
	}
}

func getOtelCollectorConfigResourceVersion(dk *dynakube.DynaKube, resourceVersion *string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		otelCollectorConfig, err := getOtelCollectorConfigMap(dk, ctx, envConfig)
		require.NoError(t, err, "failed to get otel collector config")

		*resourceVersion = otelCollectorConfig.ResourceVersion

		return ctx
	}
}

func checkOtelCollectorService(dk *dynakube.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		otelCollectorService, err := getOtelCollectorService(dk, ctx, envConfig)
		require.NoError(t, err)
		require.NotEmpty(t, otelCollectorService.Spec.Ports)

		return ctx
	}
}

func checkOtelCollectorEndpointConfigMap(dk *dynakube.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		cm, err := getOtelCollectorEndpointConfigMap(dk, ctx, envConfig)
		require.NoError(t, err)
		assert.NotNil(t, cm)

		return ctx
	}
}

func checkOtelCollectorServiceRemoved(dk *dynakube.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		_, err := getOtelCollectorService(dk, ctx, envConfig)
		require.Error(t, err)
		assert.True(t, k8serrors.IsNotFound(err), "Service still exists")

		return ctx
	}
}

func checkOtelCollectorEndpointConfigMapRemoved(dk *dynakube.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		_, err := getOtelCollectorEndpointConfigMap(dk, ctx, envConfig)
		require.Error(t, err)
		assert.True(t, k8serrors.IsNotFound(err), "Service still exists")

		return ctx
	}
}

func getOtelCollectorPodTimestamp(dk *dynakube.DynaKube, startTimestamp *time.Time) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()

		podList := pod.GetPodsForOwner(ctx, t, resources, dk.OtelCollectorStatefulsetName(), dk.Namespace)

		expectedPodCount := 1
		if dk.Spec.Templates.OpenTelemetryCollector.Replicas != nil && *dk.Spec.Templates.OpenTelemetryCollector.Replicas >= 1 {
			expectedPodCount = int(*dk.Spec.Templates.OpenTelemetryCollector.Replicas)
		}
		assert.Len(t, podList.Items, expectedPodCount)

		require.NotEmpty(t, podList.Items)
		*startTimestamp = podList.Items[0].Status.StartTime.Time

		return ctx
	}
}

func getOtelCollectorConfigMap(dk *dynakube.DynaKube, ctx context.Context, envConfig *envconf.Config) (*corev1.ConfigMap, error) {
	resources := envConfig.Client().Resources()

	var otelCollectorConfig corev1.ConfigMap
	err := resources.WithNamespace(dk.Namespace).Get(ctx, dk.Name+otelcconsts.TelemetryCollectorConfigmapSuffix, dk.Namespace, &otelCollectorConfig)

	if err != nil {
		return nil, err
	}

	return &otelCollectorConfig, nil
}

func getOtelCollectorService(dk *dynakube.DynaKube, ctx context.Context, envConfig *envconf.Config) (*corev1.Service, error) {
	resources := envConfig.Client().Resources()

	var otelCollectorService corev1.Service
	err := resources.WithNamespace(dk.Namespace).Get(ctx, dk.TelemetryIngest().GetServiceName(), dk.Namespace, &otelCollectorService)

	if err != nil {
		return nil, err
	}

	return &otelCollectorService, nil
}

func getOtelCollectorEndpointConfigMap(dk *dynakube.DynaKube, ctx context.Context, envConfig *envconf.Config) (*corev1.ConfigMap, error) {
	resources := envConfig.Client().Resources()

	var otelCollectorEndpointConfigMap corev1.ConfigMap
	err := resources.WithNamespace(dk.Namespace).Get(ctx, otelcconsts.OtlpApiEndpointConfigMapName, dk.Namespace, &otelCollectorEndpointConfigMap)

	if err != nil {
		return nil, err
	}

	return &otelCollectorEndpointConfigMap, nil
}

func waitForShutdown(name string, namespace string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()

		err := wait.For(conditions.New(resources).ResourceDeleted(&appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		}), wait.WithTimeout(10*time.Minute))

		require.NoError(t, err)

		return ctx
	}
}

func createAgTlsSecret(namespace string) (corev1.Secret, error) {
	agCrt, err := os.ReadFile(path.Join(project.TestDataDir(), consts.AgCertificate))
	if err != nil {
		return corev1.Secret{}, err
	}

	agP12, err := os.ReadFile(path.Join(project.TestDataDir(), consts.AgCertificateAndPrivateKey))
	if err != nil {
		return corev1.Secret{}, err
	}

	return secret.New(consts.AgSecretName, namespace,
		map[string][]byte{
			dynakube.TLSCertKey:                    agCrt,
			consts.AgCertificateAndPrivateKeyField: agP12,
		}), nil
}

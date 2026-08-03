// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package telemetryingest

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	otelcconsts "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/otelc/consts"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/features/consts"
	componentActiveGate "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/activegate"
	componentDynakube "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8spod"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8ssecret"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8sstatefulset"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/logs"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/tenant"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/tls"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/project"
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
		componentDynakube.WithAPIURL(secretConfig.APIURL),
		componentDynakube.WithTelemetryIngestEnabled(true),
		componentDynakube.WithOTELCollectorImageRef(t, componentDynakube.GetLatestOTELCollectorImageTagURI(t)),
	}

	testDynakube := *componentDynakube.New(options...)

	componentDynakube.Install(builder, &secretConfig, testDynakube)

	builder.Assess("otel collector started", k8sstatefulset.IsReady(testDynakube.OTELCollectorStatefulsetName(), testDynakube.Namespace))
	builder.Assess("otel collector config created", checkOTELCollectorConfig(&testDynakube))
	builder.Assess("otel collector service created", checkOTELCollectorService(&testDynakube))

	return builder.Feature()
}

// Rollout of OTel collector and a local in-cluster ActiveGate. Make sure that components are cleaned up after telemetryIngest gets disabled.
func WithLocalActiveGateAndCleanup(t *testing.T) features.Feature {
	builder := features.New("telemetryingest-with-local-active-gate-component-rollout-and-cleanup-after-disable")

	secretConfig := tenant.GetSingleTenantSecret(t)

	optionsTelemetryIngestEnabled := []componentDynakube.Option{
		componentDynakube.WithAPIURL(secretConfig.APIURL),
		componentDynakube.WithTelemetryIngestEnabled(true, "zipkin"),
		componentDynakube.WithOTELCollectorImageRef(t, componentDynakube.GetLatestOTELCollectorImageTagURI(t)),
		componentDynakube.WithActiveGateModules(activegate.KubeMonCapability.DisplayName),
		componentDynakube.WithActiveGateTLSSecret(consts.AgSecretName),
	}

	testDynakube := *componentDynakube.New(optionsTelemetryIngestEnabled...)

	agSecret, err := createAgTLSSecret(testDynakube.Namespace)
	require.NoError(t, err, "failed to create ag-tls secret")
	builder.Assess("create AG TLS secret", k8ssecret.Create(agSecret))

	componentDynakube.Install(builder, &secretConfig, testDynakube)
	builder.Assess("active gate pod is running", checkActiveGateContainer(&testDynakube))

	builder.Assess("otel collector started", k8sstatefulset.IsReady(testDynakube.OTELCollectorStatefulsetName(), testDynakube.Namespace))
	builder.Assess("otel collector config created", checkOTELCollectorConfig(&testDynakube))
	builder.Assess("otel collector service created", checkOTELCollectorService(&testDynakube))
	builder.Assess("otel collector endpoint configmap created", checkOTELCollectorEndpointConfigMap(&testDynakube))

	optionsTelemetryIngestDisabled := []componentDynakube.Option{
		componentDynakube.WithAPIURL(secretConfig.APIURL),
		componentDynakube.WithTelemetryIngestEnabled(false),
		componentDynakube.WithOTELCollectorImageRef(t, componentDynakube.GetLatestOTELCollectorImageTagURI(t)),
		componentDynakube.WithActiveGateModules(activegate.KubeMonCapability.DisplayName),
		componentDynakube.WithActiveGateTLSSecret(consts.AgSecretName),
	}

	testDynakubeNoTelemetryIngest := *componentDynakube.New(optionsTelemetryIngestDisabled...)
	componentDynakube.Update(builder, testDynakubeNoTelemetryIngest)

	builder.Assess("otel collector shutdown", waitForShutdown(testDynakubeNoTelemetryIngest.OTELCollectorStatefulsetName(), testDynakubeNoTelemetryIngest.Namespace))
	builder.Assess("otel collector config removed", checkOTELCollectorConfigRemoved(&testDynakubeNoTelemetryIngest))
	builder.Assess("otel collector service removed", checkOTELCollectorServiceRemoved(&testDynakubeNoTelemetryIngest))
	builder.Assess("otel collector endpoint configmap removed", checkOTELCollectorEndpointConfigMapRemoved(&testDynakubeNoTelemetryIngest))

	return builder.Feature()
}

// Rollout of OTel collector with TLS secret to secure the telemetryIngest endpoints
func WithTelemetryIngestEndpointTLS(t *testing.T) features.Feature {
	builder := features.New("telemetryingest-with-otel-collector-endpoint-tls")

	secretConfig := tenant.GetSingleTenantSecret(t)

	options := []componentDynakube.Option{
		componentDynakube.WithAPIURL(secretConfig.APIURL),
		componentDynakube.WithTelemetryIngestEnabled(true),
		componentDynakube.WithOTELCollectorImageRef(t, componentDynakube.GetLatestOTELCollectorImageTagURI(t)),
		componentDynakube.WithTelemetryIngestEndpointTLS(consts.TelemetryIngestTLSSecretName),
	}

	testDynakube := *componentDynakube.New(options...)

	tlsSecret, err := tls.CreateTestdataTLSSecret(testDynakube.Namespace, consts.TelemetryIngestTLSSecretName, TelemetryIngestTLSKey, TelemetryIngestTLSCrt)
	require.NoError(t, err, "failed to create TLS secret for otel collector endpoints")

	builder.Assess("create OTel collector endpoint TLS secret", k8ssecret.Create(tlsSecret))

	componentDynakube.Install(builder, &secretConfig, testDynakube)

	builder.Assess("otel collector started", k8sstatefulset.IsReady(testDynakube.OTELCollectorStatefulsetName(), testDynakube.Namespace))
	builder.Assess("otel collector config created", checkOTELCollectorConfig(&testDynakube))
	builder.Assess("otel collector service created", checkOTELCollectorService(&testDynakube))
	builder.Assess("otel collector endpoint configmap created", checkOTELCollectorEndpointConfigMap(&testDynakube))

	builder.WithTeardown("deleted OTel collector endpoint TLS secret", k8ssecret.Delete(tlsSecret))

	return builder.Feature()
}

// Make sure the Otel collector configuration is updated and pods are restarted when protocols for telemetryIngest change
func OTELCollectorConfigUpdate(t *testing.T) features.Feature {
	builder := features.New("telemetryingest-configuration-update")

	secretConfig := tenant.GetSingleTenantSecret(t)

	optionsZipkin := []componentDynakube.Option{
		componentDynakube.WithAPIURL(secretConfig.APIURL),
		componentDynakube.WithTelemetryIngestEnabled(true, "zipkin"),
		componentDynakube.WithOTELCollectorImageRef(t, componentDynakube.GetLatestOTELCollectorImageTagURI(t)),
	}

	testDynakubeZipkin := *componentDynakube.New(optionsZipkin...)

	componentDynakube.Install(builder, &secretConfig, testDynakubeZipkin)

	builder.Assess("otel collector started", k8sstatefulset.IsReady(testDynakubeZipkin.OTELCollectorStatefulsetName(), testDynakubeZipkin.Namespace))
	builder.Assess("otel collector config created", checkOTELCollectorConfig(&testDynakubeZipkin))
	builder.Assess("otel collector service created", checkOTELCollectorService(&testDynakubeZipkin))
	builder.Assess("otel collector endpoint configmap created", checkOTELCollectorEndpointConfigMap(&testDynakubeZipkin))

	var zipkinConfigResourceVersion string
	builder.Assess("otel collector zipkin configuration timestamp", getOTELCollectorConfigResourceVersion(&testDynakubeZipkin, &zipkinConfigResourceVersion))

	var zipkinPodStartTS time.Time
	builder.Assess("otel collector zipkin pod creation timestamp", getOTELCollectorPodTimestamp(&testDynakubeZipkin, &zipkinPodStartTS))

	optionsJaeger := []componentDynakube.Option{
		componentDynakube.WithAPIURL(secretConfig.APIURL),
		componentDynakube.WithTelemetryIngestEnabled(true, "jaeger"),
		componentDynakube.WithOTELCollectorImageRef(t, componentDynakube.GetLatestOTELCollectorImageTagURI(t)),
	}

	testDynakubeJaeger := *componentDynakube.New(optionsJaeger...)
	componentDynakube.Update(builder, testDynakubeJaeger)

	builder.Assess("otel collector updated", k8sstatefulset.WaitFor(testDynakubeJaeger.OTELCollectorStatefulsetName(), testDynakubeJaeger.Namespace))
	builder.Assess("otel collector config updated", checkOTELCollectorConfig(&testDynakubeJaeger))
	builder.Assess("otel collector service updated", checkOTELCollectorService(&testDynakubeJaeger))

	var jaegerConfigResourceVersion string
	builder.Assess("otel collector jaeger configuration timestamp", getOTELCollectorConfigResourceVersion(&testDynakubeJaeger, &jaegerConfigResourceVersion))
	builder.Assess("otel collector jaeger configuration updated", func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
		assert.NotEqual(t, jaegerConfigResourceVersion, zipkinConfigResourceVersion)

		return ctx
	})

	var jaegerPodStartTS time.Time
	builder.Assess("otel collector jaeger pod creation timestamp", getOTELCollectorPodTimestamp(&testDynakubeJaeger, &jaegerPodStartTS))
	builder.Assess("otel collector jaeger pod restarted", func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
		assert.Greater(t, jaegerPodStartTS, zipkinPodStartTS)

		return ctx
	})

	return builder.Feature()
}

func checkActiveGateContainer(dk *dynakube.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		componentActiveGate.CheckContainer(dk)

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
	require.NotEmpty(t, tail, "list of AG active modules not found")

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

func checkOTELCollectorConfig(dk *dynakube.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		otelCollectorConfig, err := getOTELCollectorConfigMap(dk, ctx, envConfig)
		require.NoError(t, err, "failed to get otel collector config")

		require.NotNil(t, otelCollectorConfig.Data)

		return ctx
	}
}

func checkOTELCollectorConfigRemoved(dk *dynakube.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		_, err := getOTELCollectorConfigMap(dk, ctx, envConfig)
		require.Error(t, err)
		assert.True(t, k8serrors.IsNotFound(err), "ConfigMap still exists")

		return ctx
	}
}

func getOTELCollectorConfigResourceVersion(dk *dynakube.DynaKube, resourceVersion *string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		otelCollectorConfig, err := getOTELCollectorConfigMap(dk, ctx, envConfig)
		require.NoError(t, err, "failed to get otel collector config")

		*resourceVersion = otelCollectorConfig.ResourceVersion

		return ctx
	}
}

func checkOTELCollectorService(dk *dynakube.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		otelCollectorService, err := getOTELCollectorService(dk, ctx, envConfig)
		require.NoError(t, err)
		require.NotEmpty(t, otelCollectorService.Spec.Ports)

		return ctx
	}
}

func checkOTELCollectorEndpointConfigMap(dk *dynakube.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		cm, err := getOTELCollectorEndpointConfigMap(dk, ctx, envConfig)
		require.NoError(t, err)
		assert.NotNil(t, cm)

		return ctx
	}
}

func checkOTELCollectorServiceRemoved(dk *dynakube.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		_, err := getOTELCollectorService(dk, ctx, envConfig)
		require.Error(t, err)
		assert.True(t, k8serrors.IsNotFound(err), "Service still exists")

		return ctx
	}
}

func checkOTELCollectorEndpointConfigMapRemoved(dk *dynakube.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		_, err := getOTELCollectorEndpointConfigMap(dk, ctx, envConfig)
		require.Error(t, err)
		assert.True(t, k8serrors.IsNotFound(err), "Service still exists")

		return ctx
	}
}

func getOTELCollectorPodTimestamp(dk *dynakube.DynaKube, startTimestamp *time.Time) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()

		podList := k8spod.ListForOwner(ctx, t, resources, dk.OTELCollectorStatefulsetName(), dk.Namespace)

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

func getOTELCollectorConfigMap(dk *dynakube.DynaKube, ctx context.Context, envConfig *envconf.Config) (*corev1.ConfigMap, error) {
	resources := envConfig.Client().Resources()

	var otelCollectorConfig corev1.ConfigMap
	err := resources.WithNamespace(dk.Namespace).Get(ctx, dk.Name+otelcconsts.TelemetryCollectorConfigmapSuffix, dk.Namespace, &otelCollectorConfig)

	if err != nil {
		return nil, err
	}

	return &otelCollectorConfig, nil
}

func getOTELCollectorService(dk *dynakube.DynaKube, ctx context.Context, envConfig *envconf.Config) (*corev1.Service, error) {
	resources := envConfig.Client().Resources()

	var otelCollectorService corev1.Service
	err := resources.WithNamespace(dk.Namespace).Get(ctx, dk.TelemetryIngest().GetServiceName(), dk.Namespace, &otelCollectorService)

	if err != nil {
		return nil, err
	}

	return &otelCollectorService, nil
}

func getOTELCollectorEndpointConfigMap(dk *dynakube.DynaKube, ctx context.Context, envConfig *envconf.Config) (*corev1.ConfigMap, error) {
	resources := envConfig.Client().Resources()

	var otelCollectorEndpointConfigMap corev1.ConfigMap
	err := resources.WithNamespace(dk.Namespace).Get(ctx, otelcconsts.OTLPAPIEndpointConfigMapName, dk.Namespace, &otelCollectorEndpointConfigMap)

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

func createAgTLSSecret(namespace string) (corev1.Secret, error) {
	agCrt, err := os.ReadFile(filepath.Join(project.TestDataDir(), consts.AgCertificate))
	if err != nil {
		return corev1.Secret{}, err
	}

	agP12, err := os.ReadFile(filepath.Join(project.TestDataDir(), consts.AgCertificateAndPrivateKey))
	if err != nil {
		return corev1.Secret{}, err
	}

	return k8ssecret.New(consts.AgSecretName, namespace,
		map[string][]byte{
			dynakube.ServerCertKey:                 agCrt,
			consts.AgCertificateAndPrivateKeyField: agP12,
		}), nil
}

package exporter

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/otlp"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	otelcactivegate "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/otelc/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testPodName       = "test-pod"
	testNamespaceName = "test-namespace"
	testDynakubeName  = "test-dynakube"
)

func TestMutator_IsEnabled(t *testing.T) {
	t.Run("auto-injection disabled in DynaKube", func(t *testing.T) {
		dk := getTestDynakube()

		dk.Spec.OTLPExporterConfiguration = nil

		request := createTestMutationRequest(t, dk)

		m := Mutator{}

		require.False(t, m.IsEnabled(request.BaseRequest))
	})
	t.Run("auto-injection disabled via DynaKube annotations", func(t *testing.T) {
		dk := getTestDynakube()

		dk.Annotations[exp.InjectionAutomaticKey] = "false"

		request := createTestMutationRequest(t, dk)

		m := Mutator{}

		require.False(t, m.IsEnabled(request.BaseRequest))
	})
	t.Run("auto-injection disabled on pod", func(t *testing.T) {
		dk := getTestDynakube()
		request := createTestMutationRequest(t, dk)

		request.Pod.Annotations[mutator.AnnotationDynatraceInject] = "false"

		m := Mutator{}

		require.False(t, m.IsEnabled(request.BaseRequest))
	})
	t.Run("auto-injection enabled via 'feature.dynatrace.com/automatic-injection' default feature flag value", func(t *testing.T) {
		dk := getTestDynakube()
		request := createTestMutationRequest(t, dk)

		m := Mutator{}

		require.True(t, m.IsEnabled(request.BaseRequest))
	})
	t.Run("auto-injection enabled via 'otlp-exporter-configuration.dynatrace.com/inject' annotation on pod", func(t *testing.T) {
		dk := getTestDynakube()
		request := createTestMutationRequest(t, dk)

		request.Pod.Annotations[mutator.AnnotationOTLPInjectionEnabled] = "true"

		m := Mutator{}

		require.True(t, m.IsEnabled(request.BaseRequest))
	})
	t.Run("auto-injection enabled via 'otlp-exporter-configuration.dynatrace.com/inject' annotation on pod, namespace selector does not match", func(t *testing.T) {
		dk := getTestDynakube()

		dk.Spec.OTLPExporterConfiguration.NamespaceSelector = metav1.LabelSelector{
			MatchLabels: map[string]string{
				"otlp": "true",
			},
		}

		request := createTestMutationRequest(t, dk)

		request.Pod.Annotations[mutator.AnnotationOTLPInjectionEnabled] = "true"

		m := Mutator{}

		require.False(t, m.IsEnabled(request.BaseRequest))
	})
	t.Run("auto-injection enabled via 'otlp-exporter-configuration.dynatrace.com/inject' annotation on pod, namespace selector matches", func(t *testing.T) {
		dk := getTestDynakube()

		dk.Spec.OTLPExporterConfiguration.NamespaceSelector = metav1.LabelSelector{
			MatchLabels: map[string]string{
				"otlp": "true",
			},
		}

		request := createTestMutationRequest(t, dk)

		request.Pod.Annotations[mutator.AnnotationOTLPInjectionEnabled] = "true"

		request.Namespace.Labels = map[string]string{
			"otlp": "true",
		}

		m := Mutator{}

		require.True(t, m.IsEnabled(request.BaseRequest))
	})
}

func TestMutator_IsInjected(t *testing.T) {
	t.Run("env vars for OTLP exporter already injected", func(t *testing.T) {
		m := Mutator{}

		request := createTestMutationRequest(t, getTestDynakube())

		request.Pod.Annotations[mutator.AnnotationOTLPInjected] = "true"

		assert.True(t, m.IsInjected(request.BaseRequest))
	})
	t.Run("env vars for OTLP exporter not yed injected", func(t *testing.T) {
		m := Mutator{}

		request := createTestMutationRequest(t, getTestDynakube())

		assert.False(t, m.IsInjected(request.BaseRequest))
	})
}

func TestMutator_Mutate(t *testing.T) { //nolint:revive
	const (
		testCustomTracesEndpoint     = "http://user-endpoint/api/v2/otlp/traces"
		testCustomMetricsEndpoint    = "http://user-endpoint/api/v2/otlp/metrics"
		testCustomLogsEndpoint       = "http://user-endpoint/api/v2/otlp/logs"
		testCustomProtocol           = "grpc"
		testDynatraceTracesEndpoint  = "http://my-cluster/api/v2/otlp/v1/traces"
		testDynatraceMetricsEndpoint = "http://my-cluster/api/v2/otlp/v1/metrics"
		testDynatraceLogsEndpoint    = "http://my-cluster/api/v2/otlp/v1/logs"
		testDynatraceProtocol        = "http/protobuf"
	)

	t.Run("no OTLP exporter configuration present on DynaKube - do not modify anything", func(t *testing.T) {
		m := Mutator{}

		dk := getTestDynakube()

		dk.Spec.OTLPExporterConfiguration = nil

		request := createTestMutationRequest(t, dk)

		err := m.Mutate(request)

		require.NoError(t, err)

		containerEnvVars := request.Pod.Spec.Containers[0].Env

		assert.Empty(t, containerEnvVars)
	})
	t.Run("no OTLP ingest endpoint available - do not modify anything, return error", func(t *testing.T) {
		m := Mutator{}

		dk := getTestDynakube()

		dk.Spec.ActiveGate = activegate.Spec{
			Capabilities: []activegate.CapabilityDisplayName{
				activegate.MetricsIngestCapability.DisplayName,
			},
		}

		request := createTestMutationRequest(t, dk)

		err := m.Mutate(request)

		require.Error(t, err)

		containerEnvVars := request.Pod.Spec.Containers[0].Env

		assert.Empty(t, containerEnvVars)
	})
	t.Run("no user defined env vars present, add OTLP exporter env vars", func(t *testing.T) {
		m := Mutator{}

		request := createTestMutationRequest(t, getTestDynakube())

		request.DynaKube.Spec.APIURL = "http://my-cluster/api"

		err := m.Mutate(request)

		require.NoError(t, err)

		containerEnvVars := request.Pod.Spec.Containers[0].Env

		// verify traces exporter env vars
		assert.Contains(t, containerEnvVars, corev1.EnvVar{
			Name:  OTLPTraceEndpointEnv,
			Value: testDynatraceTracesEndpoint,
		})

		assert.Contains(t, containerEnvVars, corev1.EnvVar{
			Name:  OTLPTraceProtocolEnv,
			Value: testDynatraceProtocol,
		})

		// verify metrics exporter env vars
		assert.Contains(t, containerEnvVars, corev1.EnvVar{
			Name:  OTLPMetricsEndpointEnv,
			Value: testDynatraceMetricsEndpoint,
		})

		assert.Contains(t, containerEnvVars, corev1.EnvVar{
			Name:  OTLPMetricsProtocolEnv,
			Value: testDynatraceProtocol,
		})

		assert.Contains(t, containerEnvVars, corev1.EnvVar{
			Name:  OTLPMetricsExporterTemporalityPreference,
			Value: OTLPMetricsExporterAggregationTemporalityDelta,
		})

		// verify logs exporter env vars
		assert.Contains(t, containerEnvVars, corev1.EnvVar{
			Name:  OTLPLogsEndpointEnv,
			Value: testDynatraceLogsEndpoint,
		})

		assert.Contains(t, containerEnvVars, corev1.EnvVar{
			Name:  OTLPLogsProtocolEnv,
			Value: testDynatraceProtocol,
		})

		// verify headers env vars added with Authorization header referencing DT_API_TOKEN
		assert.Contains(t, containerEnvVars, corev1.EnvVar{Name: OTLPTraceHeadersEnv, Value: OTLPAuthorizationHeader})
		assert.Contains(t, containerEnvVars, corev1.EnvVar{Name: OTLPMetricsHeadersEnv, Value: OTLPAuthorizationHeader})
		assert.Contains(t, containerEnvVars, corev1.EnvVar{Name: OTLPLogsHeadersEnv, Value: OTLPAuthorizationHeader})

		// verify DT_API_TOKEN secret ref env var
		assertTokenEnvVarIsSet(t, containerEnvVars)
	})
	t.Run("no user defined env vars present, only metrics configured, add only metrics OTLP exporter env vars", func(t *testing.T) {
		m := Mutator{}

		dk := getTestDynakube()
		dk.Spec.OTLPExporterConfiguration = &otlp.ExporterConfigurationSpec{
			Signals: otlp.SignalConfiguration{
				Metrics: &otlp.MetricsSignal{},
			},
		}

		request := createTestMutationRequest(t, dk)

		request.DynaKube.Spec.APIURL = "http://my-cluster/api"

		err := m.Mutate(request)

		require.NoError(t, err)

		containerEnvVars := request.Pod.Spec.Containers[0].Env

		// verify traces exporter env vars not set
		assert.NotContains(t, containerEnvVars, corev1.EnvVar{
			Name:  OTLPTraceEndpointEnv,
			Value: testDynatraceTracesEndpoint,
		})

		assert.NotContains(t, containerEnvVars, corev1.EnvVar{
			Name:  OTLPTraceProtocolEnv,
			Value: testDynatraceProtocol,
		})

		// verify metrics exporter env vars
		assert.Contains(t, containerEnvVars, corev1.EnvVar{
			Name:  OTLPMetricsEndpointEnv,
			Value: testDynatraceMetricsEndpoint,
		})

		assert.Contains(t, containerEnvVars, corev1.EnvVar{
			Name:  OTLPMetricsProtocolEnv,
			Value: testDynatraceProtocol,
		})

		// verify logs exporter env vars not set
		assert.NotContains(t, containerEnvVars, corev1.EnvVar{
			Name:  OTLPLogsEndpointEnv,
			Value: testDynatraceLogsEndpoint,
		})

		assert.NotContains(t, containerEnvVars, corev1.EnvVar{
			Name:  OTLPLogsProtocolEnv,
			Value: testDynatraceProtocol,
		})

		// verify headers env vars added with Authorization header referencing DT_API_TOKEN
		assert.NotContains(t, containerEnvVars, corev1.EnvVar{Name: OTLPTraceHeadersEnv, Value: OTLPAuthorizationHeader})
		assert.Contains(t, containerEnvVars, corev1.EnvVar{Name: OTLPMetricsHeadersEnv, Value: OTLPAuthorizationHeader})
		assert.NotContains(t, containerEnvVars, corev1.EnvVar{Name: OTLPLogsHeadersEnv, Value: OTLPAuthorizationHeader})

		// verify DT_API_TOKEN secret ref env var
		assertTokenEnvVarIsSet(t, containerEnvVars)
	})
	t.Run("user defined env vars present, do not add OTLP exporter env vars", func(t *testing.T) {
		m := Mutator{}

		request := createTestMutationRequest(t, getTestDynakube())

		request.DynaKube.Spec.APIURL = "http://my-cluster/api"

		request.Pod.Spec.Containers[0].Env = []corev1.EnvVar{
			{
				Name:  OTLPTraceEndpointEnv,
				Value: testCustomTracesEndpoint,
			},
			{
				Name:  OTLPTraceProtocolEnv,
				Value: testCustomProtocol,
			},
			{
				Name:  OTLPMetricsEndpointEnv,
				Value: testCustomMetricsEndpoint,
			},
			{
				Name:  OTLPMetricsProtocolEnv,
				Value: testCustomProtocol,
			},
			{
				Name:  OTLPLogsEndpointEnv,
				Value: testCustomLogsEndpoint,
			},
			{
				Name:  OTLPLogsProtocolEnv,
				Value: testCustomProtocol,
			},
		}

		err := m.Mutate(request)

		require.NoError(t, err)

		containerEnvVars := request.Pod.Spec.Containers[0].Env

		// verify traces exporter env vars
		assert.Contains(t, containerEnvVars, corev1.EnvVar{
			Name:  OTLPTraceEndpointEnv,
			Value: testCustomTracesEndpoint,
		})

		assert.Contains(t, containerEnvVars, corev1.EnvVar{
			Name:  OTLPTraceProtocolEnv,
			Value: testCustomProtocol,
		})

		// verify metrics exporter env vars
		assert.Contains(t, containerEnvVars, corev1.EnvVar{
			Name:  OTLPMetricsEndpointEnv,
			Value: testCustomMetricsEndpoint,
		})

		assert.Contains(t, containerEnvVars, corev1.EnvVar{
			Name:  OTLPMetricsProtocolEnv,
			Value: testCustomProtocol,
		})

		// verify logs exporter env vars
		assert.Contains(t, containerEnvVars, corev1.EnvVar{
			Name:  OTLPLogsEndpointEnv,
			Value: testCustomLogsEndpoint,
		})

		assert.Contains(t, containerEnvVars, corev1.EnvVar{
			Name:  OTLPLogsProtocolEnv,
			Value: testCustomProtocol,
		})

		// verify no headers or token env vars were added due to skip
		assert.False(t, k8senv.Contains(containerEnvVars, OTLPTraceHeadersEnv))
		assert.False(t, k8senv.Contains(containerEnvVars, OTLPMetricsHeadersEnv))
		assert.False(t, k8senv.Contains(containerEnvVars, OTLPLogsHeadersEnv))
		assert.False(t, k8senv.Contains(containerEnvVars, DynatraceAPITokenEnv))
	})
	t.Run("general otlp exporter user defined env vars present, do not add specific OTLP exporter env vars", func(t *testing.T) {
		m := Mutator{}

		request := createTestMutationRequest(t, getTestDynakube())

		request.DynaKube.Spec.APIURL = "http://my-cluster/api"

		request.Pod.Spec.Containers[0].Env = []corev1.EnvVar{
			{
				Name:  OTLPExporterEndpointEnv,
				Value: "http://user-endpoint/api/v2/otlp",
			},
			{
				Name:  OTLPExporterProtocolEnv,
				Value: testCustomProtocol,
			},
		}

		err := m.Mutate(request)

		require.NoError(t, err)

		containerEnvVars := request.Pod.Spec.Containers[0].Env

		assert.Contains(t, containerEnvVars, corev1.EnvVar{
			Name:  OTLPExporterEndpointEnv,
			Value: "http://user-endpoint/api/v2/otlp",
		})

		assert.Contains(t, containerEnvVars, corev1.EnvVar{
			Name:  OTLPExporterProtocolEnv,
			Value: testCustomProtocol,
		})

		// verify traces exporter env vars are not added
		assert.False(t, k8senv.Contains(containerEnvVars, OTLPTraceEndpointEnv))
		assert.False(t, k8senv.Contains(containerEnvVars, OTLPTraceProtocolEnv))

		// verify metrics exporter env vars are not added
		assert.False(t, k8senv.Contains(containerEnvVars, OTLPMetricsEndpointEnv))
		assert.False(t, k8senv.Contains(containerEnvVars, OTLPMetricsProtocolEnv))

		// verify logs exporter env vars are not added
		assert.False(t, k8senv.Contains(containerEnvVars, OTLPLogsEndpointEnv))
		assert.False(t, k8senv.Contains(containerEnvVars, OTLPLogsProtocolEnv))

		// verify no headers or token env vars were added due to skip
		assert.False(t, k8senv.Contains(containerEnvVars, OTLPTraceHeadersEnv))
		assert.False(t, k8senv.Contains(containerEnvVars, OTLPMetricsHeadersEnv))
		assert.False(t, k8senv.Contains(containerEnvVars, OTLPLogsHeadersEnv))
		assert.False(t, k8senv.Contains(containerEnvVars, DynatraceAPITokenEnv))
	})
	t.Run("general otlp exporter user defined env vars present, override enabled, add specific OTLP exporter env vars", func(t *testing.T) {
		m := Mutator{}

		request := createTestMutationRequest(t, getTestDynakube())

		override := true

		request.DynaKube.Spec.APIURL = "http://my-cluster/api"
		request.DynaKube.Spec.OTLPExporterConfiguration.OverrideEnvVars = &override
		request.DynaKube.Spec.OTLPExporterConfiguration.Signals = otlp.SignalConfiguration{
			Metrics: &otlp.MetricsSignal{},
		}

		request.Pod.Spec.Containers[0].Env = []corev1.EnvVar{
			{
				Name:  OTLPExporterEndpointEnv,
				Value: "http://user-endpoint/api/v2/otlp",
			},
			{
				Name:  OTLPExporterProtocolEnv,
				Value: testCustomProtocol,
			},
		}

		err := m.Mutate(request)

		require.NoError(t, err)

		containerEnvVars := request.Pod.Spec.Containers[0].Env

		assert.Contains(t, containerEnvVars, corev1.EnvVar{
			Name:  OTLPExporterEndpointEnv,
			Value: "http://user-endpoint/api/v2/otlp",
		})

		assert.Contains(t, containerEnvVars, corev1.EnvVar{
			Name:  OTLPExporterProtocolEnv,
			Value: testCustomProtocol,
		})

		// verify traces exporter env vars are not added
		assert.False(t, k8senv.Contains(containerEnvVars, OTLPTraceEndpointEnv))
		assert.False(t, k8senv.Contains(containerEnvVars, OTLPTraceProtocolEnv))

		// verify metrics exporter env vars are added
		assert.Contains(t, containerEnvVars, corev1.EnvVar{
			Name:  OTLPMetricsEndpointEnv,
			Value: testDynatraceMetricsEndpoint,
		})

		assert.Contains(t, containerEnvVars, corev1.EnvVar{
			Name:  OTLPMetricsProtocolEnv,
			Value: testDynatraceProtocol,
		})

		// headers for metrics are added, traces/logs are not
		assert.Contains(t, containerEnvVars, corev1.EnvVar{Name: OTLPMetricsHeadersEnv, Value: OTLPAuthorizationHeader})
		assert.False(t, k8senv.Contains(containerEnvVars, OTLPTraceHeadersEnv))
		assert.False(t, k8senv.Contains(containerEnvVars, OTLPLogsHeadersEnv))

		// verify DT_API_TOKEN secret ref env var
		assertTokenEnvVarIsSet(t, containerEnvVars)
	})
	t.Run("specific otlp exporter user defined env vars present, override disabled, do not add specific OTLP exporter env vars", func(t *testing.T) {
		m := Mutator{}

		request := createTestMutationRequest(t, getTestDynakube())

		request.DynaKube.Spec.APIURL = "http://my-cluster/api"
		request.DynaKube.Spec.OTLPExporterConfiguration.Signals = otlp.SignalConfiguration{
			Metrics: &otlp.MetricsSignal{},
		}

		request.Pod.Spec.Containers[0].Env = []corev1.EnvVar{
			{
				Name:  OTLPTraceEndpointEnv,
				Value: testCustomTracesEndpoint,
			},
			{
				Name:  OTLPTraceProtocolEnv,
				Value: testCustomProtocol,
			},
		}

		err := m.Mutate(request)

		require.NoError(t, err)

		containerEnvVars := request.Pod.Spec.Containers[0].Env

		// verify user defined env vars are kept as they are
		assert.Contains(t, containerEnvVars, corev1.EnvVar{
			Name:  OTLPTraceEndpointEnv,
			Value: testCustomTracesEndpoint,
		})

		assert.Contains(t, containerEnvVars, corev1.EnvVar{
			Name:  OTLPTraceProtocolEnv,
			Value: testCustomProtocol,
		})

		// verify metrics exporter env vars are not added
		assert.False(t, k8senv.Contains(containerEnvVars, OTLPMetricsEndpointEnv))
		assert.False(t, k8senv.Contains(containerEnvVars, OTLPMetricsProtocolEnv))

		// verify logs exporter env vars are not added
		assert.False(t, k8senv.Contains(containerEnvVars, OTLPLogsEndpointEnv))
		assert.False(t, k8senv.Contains(containerEnvVars, OTLPLogsProtocolEnv))

		// verify no headers or token env vars were added due to skip
		assert.False(t, k8senv.Contains(containerEnvVars, OTLPMetricsHeadersEnv))
		assert.False(t, k8senv.Contains(containerEnvVars, OTLPLogsHeadersEnv))
		assert.False(t, k8senv.Contains(containerEnvVars, DynatraceAPITokenEnv))
	})
	t.Run("specific otlp exporter user defined env vars present, override enabled, add other specific OTLP exporter env vars", func(t *testing.T) {
		m := Mutator{}

		request := createTestMutationRequest(t, getTestDynakube())

		override := true

		request.DynaKube.Spec.APIURL = "http://my-cluster/api"
		request.DynaKube.Spec.OTLPExporterConfiguration.OverrideEnvVars = &override
		request.DynaKube.Spec.OTLPExporterConfiguration.Signals = otlp.SignalConfiguration{
			Metrics: &otlp.MetricsSignal{},
		}

		request.Pod.Spec.Containers[0].Env = []corev1.EnvVar{
			{
				Name:  OTLPTraceEndpointEnv,
				Value: testCustomTracesEndpoint,
			},
			{
				Name:  OTLPTraceProtocolEnv,
				Value: testCustomProtocol,
			},
			{
				Name:  OTLPMetricsEndpointEnv,
				Value: testCustomMetricsEndpoint,
			},
			{
				Name:  OTLPMetricsProtocolEnv,
				Value: testCustomProtocol,
			},
		}

		err := m.Mutate(request)

		require.NoError(t, err)

		containerEnvVars := request.Pod.Spec.Containers[0].Env

		// verify user defined env vars are kept as they are
		assert.Contains(t, containerEnvVars, corev1.EnvVar{
			Name:  OTLPTraceEndpointEnv,
			Value: testCustomTracesEndpoint,
		})

		assert.Contains(t, containerEnvVars, corev1.EnvVar{
			Name:  OTLPTraceProtocolEnv,
			Value: testCustomProtocol,
		})

		// verify metrics exporter env vars are added
		assert.Contains(t, containerEnvVars, corev1.EnvVar{
			Name:  OTLPMetricsEndpointEnv,
			Value: testDynatraceMetricsEndpoint,
		})

		assert.Contains(t, containerEnvVars, corev1.EnvVar{
			Name:  OTLPMetricsProtocolEnv,
			Value: testDynatraceProtocol,
		})

		// verify logs exporter env vars are not added
		assert.False(t, k8senv.Contains(containerEnvVars, OTLPLogsEndpointEnv))
		assert.False(t, k8senv.Contains(containerEnvVars, OTLPLogsProtocolEnv))
	})
	t.Run("one specific otlp exporter user defined env vars present, override disabled, do not add other specific OTLP exporter env vars", func(t *testing.T) {
		m := Mutator{}

		request := createTestMutationRequest(t, getTestDynakube())

		request.DynaKube.Spec.APIURL = "http://my-cluster/api"
		request.DynaKube.Spec.OTLPExporterConfiguration.Signals = otlp.SignalConfiguration{
			Metrics: &otlp.MetricsSignal{},
		}

		request.Pod.Spec.Containers[0].Env = []corev1.EnvVar{
			{
				Name:  OTLPTraceEndpointEnv,
				Value: testCustomTracesEndpoint,
			},
			{
				Name:  OTLPTraceProtocolEnv,
				Value: testCustomProtocol,
			},
		}

		err := m.Mutate(request)

		require.NoError(t, err)

		containerEnvVars := request.Pod.Spec.Containers[0].Env

		// verify user defined env vars are kept as they are
		assert.Contains(t, containerEnvVars, corev1.EnvVar{
			Name:  OTLPTraceEndpointEnv,
			Value: testCustomTracesEndpoint,
		})

		assert.Contains(t, containerEnvVars, corev1.EnvVar{
			Name:  OTLPTraceProtocolEnv,
			Value: testCustomProtocol,
		})

		// verify metrics exporter env vars are not added
		assert.False(t, k8senv.Contains(containerEnvVars, OTLPMetricsEndpointEnv))
		assert.False(t, k8senv.Contains(containerEnvVars, OTLPMetricsProtocolEnv))

		// verify logs exporter env vars are not added
		assert.False(t, k8senv.Contains(containerEnvVars, OTLPLogsEndpointEnv))
		assert.False(t, k8senv.Contains(containerEnvVars, OTLPLogsProtocolEnv))
	})
	t.Run("activegate with ca cert and override enabled -> mounts cert volume and injects certificate env vars", func(t *testing.T) {
		m := Mutator{}

		dk := getTestDynakube()
		dk.Spec.APIURL = "http://my-cluster/api"
		// Enable ActiveGate capability + TLS secret so HasCaCert() is true
		dk.Spec.ActiveGate = activegate.Spec{Capabilities: []activegate.CapabilityDisplayName{activegate.MetricsIngestCapability.DisplayName}, TLSSecretName: "custom-tls-secret"}
		dk.Status.OneAgent.ConnectionInfo.TenantUUID = "dummy-uuid"

		override := true
		dk.Spec.OTLPExporterConfiguration.OverrideEnvVars = &override

		request := createTestMutationRequest(t, dk)
		err := m.Mutate(request)
		require.NoError(t, err)

		// Volume referencing certificate secret must be present
		volFound := false
		for _, v := range request.Pod.Spec.Volumes {
			if v.Name == activeGateTrustedCertVolumeName {
				volFound = true
				require.NotNil(t, v.Secret)
				assert.Equal(t, consts.OTLPExporterCertsSecretName, v.Secret.SecretName)

				break
			}
		}
		assert.True(t, volFound, "expected cert volume")

		certPath := getCertificatePath()
		for _, c := range request.Pod.Spec.Containers {
			mountFound := false
			for _, mnt := range c.VolumeMounts {
				if mnt.Name == activeGateTrustedCertVolumeName && mnt.MountPath == exporterCertsMountPath {
					mountFound = true
					assert.True(t, mnt.ReadOnly)

					break
				}
			}
			assert.True(t, mountFound, "expected cert mount on container %s", c.Name)

			// Certificate env vars injected because addCertificate=true passed to injectors
			for _, e := range c.Env {
				if e.Name == OTLPTraceCertificateEnv || e.Name == OTLPMetricsCertificateEnv || e.Name == OTLPLogsCertificateEnv {
					assert.Equal(t, certPath, e.Value)
				}
			}

			assert.True(t, k8senv.Contains(c.Env, OTLPTraceCertificateEnv))
			assert.True(t, k8senv.Contains(c.Env, OTLPMetricsCertificateEnv))
			assert.True(t, k8senv.Contains(c.Env, OTLPLogsCertificateEnv))

			// assert NO_PROXY is set
			noProxyEnv := k8senv.Find(c.Env, NoProxyEnv)
			require.NotNil(t, noProxyEnv)
			assert.Equal(t, otelcactivegate.GetServiceFQDN(&request.DynaKube), noProxyEnv.Value)
		}
		// Init containers should not have the mount
		for _, c := range request.Pod.Spec.InitContainers {
			for _, mnt := range c.VolumeMounts {
				assert.NotEqual(t, activeGateTrustedCertVolumeName, mnt.Name)
			}
		}
	})
}

func assertTokenEnvVarIsSet(t *testing.T, containerEnvVars []corev1.EnvVar) {
	var dtTokenVar *corev1.EnvVar
	for i := range containerEnvVars {
		if containerEnvVars[i].Name == DynatraceAPITokenEnv {
			dtTokenVar = &containerEnvVars[i]

			break
		}
	}
	require.NotNil(t, dtTokenVar, "expected DT_API_TOKEN env var to be injected")
	require.NotNil(t, dtTokenVar.ValueFrom)
	require.NotNil(t, dtTokenVar.ValueFrom.SecretKeyRef)
	assert.Equal(t, consts.OTLPExporterSecretName, dtTokenVar.ValueFrom.SecretKeyRef.Name)
	assert.Equal(t, dynatrace.DataIngestToken, dtTokenVar.ValueFrom.SecretKeyRef.Key)
}

func TestMutator_Reinvoke(t *testing.T) {
	t.Run("return true if pod is modified", func(t *testing.T) {
		m := Mutator{}

		request := createTestMutationRequest(t, getTestDynakube())

		mutated := m.Reinvoke(request.ToReinvocationRequest())

		require.True(t, mutated)
	})
	t.Run("return false if pod is not modified", func(t *testing.T) {
		m := Mutator{}

		dk := getTestDynakube()

		dk.Spec.OTLPExporterConfiguration = nil

		request := createTestMutationRequest(t, dk)

		mutated := m.Reinvoke(request.ToReinvocationRequest())

		require.False(t, mutated)
	})
}

func Test_ensureCertificateVolumeMounted(t *testing.T) {
	newContainer := func() corev1.Container { return corev1.Container{Name: "app"} }

	t.Run("adds mount when absent", func(t *testing.T) {
		c := newContainer()
		require.Empty(t, c.VolumeMounts)
		ensureCertificateVolumeMounted(&c)
		require.Len(t, c.VolumeMounts, 1)
		vm := c.VolumeMounts[0]
		assert.Equal(t, activeGateTrustedCertVolumeName, vm.Name)
		assert.Equal(t, exporterCertsMountPath, vm.MountPath)
		assert.True(t, vm.ReadOnly)
	})

	t.Run("does not duplicate mount", func(t *testing.T) {
		c := newContainer()
		c.VolumeMounts = []corev1.VolumeMount{{Name: activeGateTrustedCertVolumeName, MountPath: exporterCertsMountPath, ReadOnly: true}}
		ensureCertificateVolumeMounted(&c)
		assert.Len(t, c.VolumeMounts, 1)
	})
}

func Test_addActiveGateCertVolume(t *testing.T) {
	newPod := func() *corev1.Pod { return &corev1.Pod{} }
	baseDK := func() dynakube.DynaKube { return dynakube.DynaKube{} }

	t.Run("no activegate -> no volume", func(t *testing.T) {
		dk := baseDK() // ActiveGate not enabled
		pod := newPod()
		addActiveGateCertVolume(dk, pod)
		assert.Empty(t, pod.Spec.Volumes)
	})

	t.Run("activegate with cert secret -> volume added once", func(t *testing.T) {
		dk := baseDK()
		dk.Spec.ActiveGate = activegate.Spec{Capabilities: []activegate.CapabilityDisplayName{activegate.DynatraceAPICapability.DisplayName}, TLSSecretName: "custom-tls"}
		pod := newPod()
		addActiveGateCertVolume(dk, pod)
		require.Len(t, pod.Spec.Volumes, 1)
		v := pod.Spec.Volumes[0]
		assert.Equal(t, activeGateTrustedCertVolumeName, v.Name)
		require.NotNil(t, v.Secret)
		assert.Equal(t, consts.OTLPExporterCertsSecretName, v.Secret.SecretName)

		// second call should not duplicate
		addActiveGateCertVolume(dk, pod)
		assert.Len(t, pod.Spec.Volumes, 1)
	})
}

func createTestMutationRequest(t *testing.T, dk *dynakube.DynaKube) *mutator.MutationRequest {
	return mutator.NewMutationRequest(t.Context(), *getTestNamespace(), nil, getTestPod(), *dk)
}

func getTestNamespace() *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNamespaceName,
			Labels: map[string]string{
				mutator.InjectionInstanceLabel: testDynakubeName,
			},
		},
	}
}

func getTestDynakube() *dynakube.DynaKube {
	return &dynakube.DynaKube{
		ObjectMeta: getTestDynakubeMeta(),
		Spec: dynakube.DynaKubeSpec{
			OTLPExporterConfiguration: &otlp.ExporterConfigurationSpec{
				Signals: otlp.SignalConfiguration{
					Metrics: &otlp.MetricsSignal{},
					Traces:  &otlp.TracesSignal{},
					Logs:    &otlp.LogsSignal{},
				},
			},
		},
	}
}

func getTestDynakubeMeta() metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:        testDynakubeName,
		Namespace:   testNamespaceName,
		Annotations: map[string]string{},
	}
}

func getTestPod() *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        testPodName,
			Namespace:   testNamespaceName,
			Annotations: map[string]string{},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "container",
					Image: "alpine",
				},
			},
			InitContainers: []corev1.Container{
				{
					Name:  "init-container",
					Image: "alpine",
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "volume",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
		},
	}
}

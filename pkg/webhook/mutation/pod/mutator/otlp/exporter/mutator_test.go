package exporter

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/otlpexporterconfiguration"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
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

		request.Pod.Annotations[AnnotationInject] = "true"

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

		request.Pod.Annotations[AnnotationInject] = "true"

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

		request.Pod.Annotations[AnnotationInject] = "true"

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

		request.Pod.Annotations[AnnotationInjected] = "true"

		assert.True(t, m.IsInjected(request.BaseRequest))
	})
	t.Run("env vars for OTLP exporter not yed injected", func(t *testing.T) {
		m := Mutator{}

		request := createTestMutationRequest(t, getTestDynakube())

		assert.False(t, m.IsInjected(request.BaseRequest))
	})
}

func TestMutator_Mutate(t *testing.T) { //nolint:revive
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
			Value: "http://my-cluster/api/v2/otlp/traces",
		})

		assert.Contains(t, containerEnvVars, corev1.EnvVar{
			Name:  OTLPTraceProtocolEnv,
			Value: "http/protobuf",
		})

		// verify metrics exporter env vars
		assert.Contains(t, containerEnvVars, corev1.EnvVar{
			Name:  OTLPMetricsEndpointEnv,
			Value: "http://my-cluster/api/v2/otlp/metrics",
		})

		assert.Contains(t, containerEnvVars, corev1.EnvVar{
			Name:  OTLPMetricsProtocolEnv,
			Value: "http/protobuf",
		})

		// verify logs exporter env vars
		assert.Contains(t, containerEnvVars, corev1.EnvVar{
			Name:  OTLPLogsEndpointEnv,
			Value: "http://my-cluster/api/v2/otlp/logs",
		})

		assert.Contains(t, containerEnvVars, corev1.EnvVar{
			Name:  OTLPLogsProtocolEnv,
			Value: "http/protobuf",
		})

		// verify headers env vars added with Authorization header referencing DT_API_TOKEN
		assert.Contains(t, containerEnvVars, corev1.EnvVar{Name: OTLPTraceHeadersEnv, Value: OTLPAuthorizationHeader})
		assert.Contains(t, containerEnvVars, corev1.EnvVar{Name: OTLPMetricsHeadersEnv, Value: OTLPAuthorizationHeader})
		assert.Contains(t, containerEnvVars, corev1.EnvVar{Name: OTLPLogsHeadersEnv, Value: OTLPAuthorizationHeader})

		// verify DT_API_TOKEN secret ref env var
		var dtTokenVar *corev1.EnvVar
		for i := range containerEnvVars {
			if containerEnvVars[i].Name == DynatraceAPIToken {
				dtTokenVar = &containerEnvVars[i]
				break
			}
		}
		require.NotNil(t, dtTokenVar, "expected DT_API_TOKEN env var to be injected")
		require.NotNil(t, dtTokenVar.ValueFrom)
		require.NotNil(t, dtTokenVar.ValueFrom.SecretKeyRef)
		assert.Equal(t, consts.OTLPExporterSecretName, dtTokenVar.ValueFrom.SecretKeyRef.Name)
		assert.Equal(t, dynatrace.DataIngestToken, dtTokenVar.ValueFrom.SecretKeyRef.Key)
	})
	t.Run("user defined env vars present, do not add OTLP exporter env vars", func(t *testing.T) {
		m := Mutator{}

		request := createTestMutationRequest(t, getTestDynakube())

		request.DynaKube.Spec.APIURL = "http://my-cluster/api"

		request.Pod.Spec.Containers[0].Env = []corev1.EnvVar{
			{
				Name:  OTLPTraceEndpointEnv,
				Value: "http://user-endpoint/api/v2/otlp/traces",
			},
			{
				Name:  OTLPTraceProtocolEnv,
				Value: "grpc",
			},
			{
				Name:  OTLPMetricsEndpointEnv,
				Value: "http://user-endpoint/api/v2/otlp/metrics",
			},
			{
				Name:  OTLPMetricsProtocolEnv,
				Value: "grpc",
			},
			{
				Name:  OTLPLogsEndpointEnv,
				Value: "http://user-endpoint/api/v2/otlp/logs",
			},
			{
				Name:  OTLPLogsProtocolEnv,
				Value: "grpc",
			},
		}

		err := m.Mutate(request)

		require.NoError(t, err)

		containerEnvVars := request.Pod.Spec.Containers[0].Env

		// verify traces exporter env vars
		assert.Contains(t, containerEnvVars, corev1.EnvVar{
			Name:  OTLPTraceEndpointEnv,
			Value: "http://user-endpoint/api/v2/otlp/traces",
		})

		assert.Contains(t, containerEnvVars, corev1.EnvVar{
			Name:  OTLPTraceProtocolEnv,
			Value: "grpc",
		})

		// verify metrics exporter env vars
		assert.Contains(t, containerEnvVars, corev1.EnvVar{
			Name:  OTLPMetricsEndpointEnv,
			Value: "http://user-endpoint/api/v2/otlp/metrics",
		})

		assert.Contains(t, containerEnvVars, corev1.EnvVar{
			Name:  OTLPMetricsProtocolEnv,
			Value: "grpc",
		})

		// verify logs exporter env vars
		assert.Contains(t, containerEnvVars, corev1.EnvVar{
			Name:  OTLPLogsEndpointEnv,
			Value: "http://user-endpoint/api/v2/otlp/logs",
		})

		assert.Contains(t, containerEnvVars, corev1.EnvVar{
			Name:  OTLPLogsProtocolEnv,
			Value: "grpc",
		})

		// verify no headers or token env vars were added due to skip
		assert.False(t, env.IsIn(containerEnvVars, OTLPTraceHeadersEnv))
		assert.False(t, env.IsIn(containerEnvVars, OTLPMetricsHeadersEnv))
		assert.False(t, env.IsIn(containerEnvVars, OTLPLogsHeadersEnv))
		assert.False(t, env.IsIn(containerEnvVars, DynatraceAPIToken))
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
				Value: "grpc",
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
			Value: "grpc",
		})

		// verify traces exporter env vars are not added
		assert.False(t, env.IsIn(containerEnvVars, OTLPTraceEndpointEnv))
		assert.False(t, env.IsIn(containerEnvVars, OTLPTraceProtocolEnv))

		// verify metrics exporter env vars are not added
		assert.False(t, env.IsIn(containerEnvVars, OTLPMetricsEndpointEnv))
		assert.False(t, env.IsIn(containerEnvVars, OTLPMetricsProtocolEnv))

		// verify logs exporter env vars are not added
		assert.False(t, env.IsIn(containerEnvVars, OTLPLogsEndpointEnv))
		assert.False(t, env.IsIn(containerEnvVars, OTLPLogsProtocolEnv))

		// verify no headers or token env vars were added due to skip
		assert.False(t, env.IsIn(containerEnvVars, OTLPTraceHeadersEnv))
		assert.False(t, env.IsIn(containerEnvVars, OTLPMetricsHeadersEnv))
		assert.False(t, env.IsIn(containerEnvVars, OTLPLogsHeadersEnv))
		assert.False(t, env.IsIn(containerEnvVars, DynatraceAPIToken))
	})
	t.Run("general otlp exporter user defined env vars present, override enabled, add specific OTLP exporter env vars", func(t *testing.T) {
		m := Mutator{}

		request := createTestMutationRequest(t, getTestDynakube())

		override := true

		request.DynaKube.Spec.APIURL = "http://my-cluster/api"
		request.DynaKube.Spec.OTLPExporterConfiguration.OverrideEnvVars = &override
		request.DynaKube.Spec.OTLPExporterConfiguration.Signals = otlpexporterconfiguration.SignalConfiguration{
			Metrics: &otlpexporterconfiguration.MetricsSignal{},
		}

		request.Pod.Spec.Containers[0].Env = []corev1.EnvVar{
			{
				Name:  OTLPExporterEndpointEnv,
				Value: "http://user-endpoint/api/v2/otlp",
			},
			{
				Name:  OTLPExporterProtocolEnv,
				Value: "grpc",
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
			Value: "grpc",
		})

		// verify traces exporter env vars are not added
		assert.False(t, env.IsIn(containerEnvVars, OTLPTraceEndpointEnv))
		assert.False(t, env.IsIn(containerEnvVars, OTLPTraceProtocolEnv))

		// verify metrics exporter env vars are added
		assert.Contains(t, containerEnvVars, corev1.EnvVar{
			Name:  OTLPMetricsEndpointEnv,
			Value: "http://my-cluster/api/v2/otlp/metrics",
		})

		assert.Contains(t, containerEnvVars, corev1.EnvVar{
			Name:  OTLPMetricsProtocolEnv,
			Value: "http/protobuf",
		})

		// headers for metrics are added, traces/logs are not
		assert.Contains(t, containerEnvVars, corev1.EnvVar{Name: OTLPMetricsHeadersEnv, Value: OTLPAuthorizationHeader})
		assert.False(t, env.IsIn(containerEnvVars, OTLPTraceHeadersEnv))
		assert.False(t, env.IsIn(containerEnvVars, OTLPLogsHeadersEnv))

		// verify DT_API_TOKEN secret ref env var
		var dtTokenVar *corev1.EnvVar
		for i := range containerEnvVars {
			if containerEnvVars[i].Name == DynatraceAPIToken {
				dtTokenVar = &containerEnvVars[i]
				break
			}
		}
		require.NotNil(t, dtTokenVar, "expected DT_API_TOKEN env var to be injected")
		require.NotNil(t, dtTokenVar.ValueFrom)
		require.NotNil(t, dtTokenVar.ValueFrom.SecretKeyRef)
		assert.Equal(t, consts.OTLPExporterSecretName, dtTokenVar.ValueFrom.SecretKeyRef.Name)
		assert.Equal(t, dynatrace.DataIngestToken, dtTokenVar.ValueFrom.SecretKeyRef.Key)
	})
	t.Run("specific otlp exporter user defined env vars present, override disabled, do not add specific OTLP exporter env vars", func(t *testing.T) {
		m := Mutator{}

		request := createTestMutationRequest(t, getTestDynakube())

		request.DynaKube.Spec.APIURL = "http://my-cluster/api"
		request.DynaKube.Spec.OTLPExporterConfiguration.Signals = otlpexporterconfiguration.SignalConfiguration{
			Metrics: &otlpexporterconfiguration.MetricsSignal{},
		}

		request.Pod.Spec.Containers[0].Env = []corev1.EnvVar{
			{
				Name:  OTLPTraceEndpointEnv,
				Value: "http://user-endpoint/api/v2/otlp/traces",
			},
			{
				Name:  OTLPTraceProtocolEnv,
				Value: "grpc",
			},
		}

		err := m.Mutate(request)

		require.NoError(t, err)

		containerEnvVars := request.Pod.Spec.Containers[0].Env

		// verify user defined env vars are kept as they are
		assert.Contains(t, containerEnvVars, corev1.EnvVar{
			Name:  OTLPTraceEndpointEnv,
			Value: "http://user-endpoint/api/v2/otlp/traces",
		})

		assert.Contains(t, containerEnvVars, corev1.EnvVar{
			Name:  OTLPTraceProtocolEnv,
			Value: "grpc",
		})

		// verify metrics exporter env vars are not added
		assert.False(t, env.IsIn(containerEnvVars, OTLPMetricsEndpointEnv))
		assert.False(t, env.IsIn(containerEnvVars, OTLPMetricsProtocolEnv))

		// verify logs exporter env vars are not added
		assert.False(t, env.IsIn(containerEnvVars, OTLPLogsEndpointEnv))
		assert.False(t, env.IsIn(containerEnvVars, OTLPLogsProtocolEnv))

		// verify no headers or token env vars were added due to skip
		assert.False(t, env.IsIn(containerEnvVars, OTLPMetricsHeadersEnv))
		assert.False(t, env.IsIn(containerEnvVars, OTLPLogsHeadersEnv))
		assert.False(t, env.IsIn(containerEnvVars, DynatraceAPIToken))
	})
	t.Run("specific otlp exporter user defined env vars present, override enabled, add other specific OTLP exporter env vars", func(t *testing.T) {
		m := Mutator{}

		request := createTestMutationRequest(t, getTestDynakube())

		override := true

		request.DynaKube.Spec.APIURL = "http://my-cluster/api"
		request.DynaKube.Spec.OTLPExporterConfiguration.OverrideEnvVars = &override
		request.DynaKube.Spec.OTLPExporterConfiguration.Signals = otlpexporterconfiguration.SignalConfiguration{
			Metrics: &otlpexporterconfiguration.MetricsSignal{},
		}

		request.Pod.Spec.Containers[0].Env = []corev1.EnvVar{
			{
				Name:  OTLPTraceEndpointEnv,
				Value: "http://user-endpoint/api/v2/otlp/traces",
			},
			{
				Name:  OTLPTraceProtocolEnv,
				Value: "grpc",
			},
			{
				Name:  OTLPMetricsEndpointEnv,
				Value: "http://user-endpoint/api/v2/otlp/metrics",
			},
			{
				Name:  OTLPMetricsProtocolEnv,
				Value: "grpc",
			},
		}

		err := m.Mutate(request)

		require.NoError(t, err)

		containerEnvVars := request.Pod.Spec.Containers[0].Env

		// verify user defined env vars are kept as they are
		assert.Contains(t, containerEnvVars, corev1.EnvVar{
			Name:  OTLPTraceEndpointEnv,
			Value: "http://user-endpoint/api/v2/otlp/traces",
		})

		assert.Contains(t, containerEnvVars, corev1.EnvVar{
			Name:  OTLPTraceProtocolEnv,
			Value: "grpc",
		})

		// verify metrics exporter env vars are added
		assert.Contains(t, containerEnvVars, corev1.EnvVar{
			Name:  OTLPMetricsEndpointEnv,
			Value: "http://my-cluster/api/v2/otlp/metrics",
		})

		assert.Contains(t, containerEnvVars, corev1.EnvVar{
			Name:  OTLPMetricsProtocolEnv,
			Value: "http/protobuf",
		})

		// verify logs exporter env vars are not added
		assert.False(t, env.IsIn(containerEnvVars, OTLPLogsEndpointEnv))
		assert.False(t, env.IsIn(containerEnvVars, OTLPLogsProtocolEnv))
	})
	t.Run("one specific otlp exporter user defined env vars present, override disabled, do not add other specific OTLP exporter env vars", func(t *testing.T) {
		m := Mutator{}

		request := createTestMutationRequest(t, getTestDynakube())

		request.DynaKube.Spec.APIURL = "http://my-cluster/api"
		request.DynaKube.Spec.OTLPExporterConfiguration.Signals = otlpexporterconfiguration.SignalConfiguration{
			Metrics: &otlpexporterconfiguration.MetricsSignal{},
		}

		request.Pod.Spec.Containers[0].Env = []corev1.EnvVar{
			{
				Name:  OTLPTraceEndpointEnv,
				Value: "http://user-endpoint/api/v2/otlp/traces",
			},
			{
				Name:  OTLPTraceProtocolEnv,
				Value: "grpc",
			},
		}

		err := m.Mutate(request)

		require.NoError(t, err)

		containerEnvVars := request.Pod.Spec.Containers[0].Env

		// verify user defined env vars are kept as they are
		assert.Contains(t, containerEnvVars, corev1.EnvVar{
			Name:  OTLPTraceEndpointEnv,
			Value: "http://user-endpoint/api/v2/otlp/traces",
		})

		assert.Contains(t, containerEnvVars, corev1.EnvVar{
			Name:  OTLPTraceProtocolEnv,
			Value: "grpc",
		})

		// verify metrics exporter env vars are not added
		assert.False(t, env.IsIn(containerEnvVars, OTLPMetricsEndpointEnv))
		assert.False(t, env.IsIn(containerEnvVars, OTLPMetricsProtocolEnv))

		// verify logs exporter env vars are not added
		assert.False(t, env.IsIn(containerEnvVars, OTLPLogsEndpointEnv))
		assert.False(t, env.IsIn(containerEnvVars, OTLPLogsProtocolEnv))
	})
	t.Run("activegate with ca cert and override enabled -> mounts cert volume and injects certificate env vars", func(t *testing.T) {
		m := Mutator{}

		dk := getTestDynakube()
		dk.Spec.APIURL = "http://my-cluster/api"
		// Enable ActiveGate capability + TLS secret so HasCaCert() is true
		dk.Spec.ActiveGate = activegate.Spec{Capabilities: []activegate.CapabilityDisplayName{activegate.MetricsIngestCapability.DisplayName}, TLSSecretName: "custom-tls-secret"}
		dk.Status.OneAgent.ConnectionInfoStatus.TenantUUID = "dummy-uuid"

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
			// Certificate env vars injected because override=true passed to injectors
			for _, e := range c.Env {
				if e.Name == OTLPTraceCertificateEnv || e.Name == OTLPMetricsCertificateEnv || e.Name == OTLPLogsCertificateEnv {
					assert.Equal(t, certPath, e.Value)
				}
			}
			assert.True(t, env.IsIn(c.Env, OTLPTraceCertificateEnv))
			assert.True(t, env.IsIn(c.Env, OTLPMetricsCertificateEnv))
			assert.True(t, env.IsIn(c.Env, OTLPLogsCertificateEnv))
		}
		// Init containers should not have the mount
		for _, c := range request.Pod.Spec.InitContainers {
			for _, mnt := range c.VolumeMounts {
				assert.NotEqual(t, activeGateTrustedCertVolumeName, mnt.Name)
			}
		}
	})
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
			OTLPExporterConfiguration: &otlpexporterconfiguration.Spec{
				Signals: otlpexporterconfiguration.SignalConfiguration{
					Metrics: &otlpexporterconfiguration.MetricsSignal{},
					Traces:  &otlpexporterconfiguration.TracesSignal{},
					Logs:    &otlpexporterconfiguration.LogsSignal{},
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

package exporter

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/otlpexporterconfiguration"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

const (
	testPodName       = "test-pod"
	testNamespaceName = "test-namespace"
	testDynakubeName  = "test-dynakube"
)

func TestMutator_IsEnabled(t *testing.T) {
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
		Name:      testDynakubeName,
		Namespace: testNamespaceName,
		Annotations: map[string]string{
			exp.OTLPExporterConfigurationKey: "true",
		},
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

func TestMutator_Mutate(t *testing.T) {
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
		assert.False(t, isEnvVarSet(containerEnvVars, OTLPTraceEndpointEnv))
		assert.False(t, isEnvVarSet(containerEnvVars, OTLPTraceProtocolEnv))

		// verify metrics exporter env vars are not added
		assert.False(t, isEnvVarSet(containerEnvVars, OTLPMetricsEndpointEnv))
		assert.False(t, isEnvVarSet(containerEnvVars, OTLPMetricsProtocolEnv))

		// verify logs exporter env vars are not added
		assert.False(t, isEnvVarSet(containerEnvVars, OTLPLogsEndpointEnv))
		assert.False(t, isEnvVarSet(containerEnvVars, OTLPLogsProtocolEnv))
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
		assert.False(t, isEnvVarSet(containerEnvVars, OTLPTraceEndpointEnv))
		assert.False(t, isEnvVarSet(containerEnvVars, OTLPTraceProtocolEnv))

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
		assert.False(t, isEnvVarSet(containerEnvVars, OTLPLogsEndpointEnv))
		assert.False(t, isEnvVarSet(containerEnvVars, OTLPLogsProtocolEnv))
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
		assert.False(t, isEnvVarSet(containerEnvVars, OTLPMetricsEndpointEnv))
		assert.False(t, isEnvVarSet(containerEnvVars, OTLPMetricsProtocolEnv))

		// verify logs exporter env vars are not added
		assert.False(t, isEnvVarSet(containerEnvVars, OTLPLogsEndpointEnv))
		assert.False(t, isEnvVarSet(containerEnvVars, OTLPLogsProtocolEnv))
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
		assert.False(t, isEnvVarSet(containerEnvVars, OTLPLogsEndpointEnv))
		assert.False(t, isEnvVarSet(containerEnvVars, OTLPLogsProtocolEnv))
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
		assert.False(t, isEnvVarSet(containerEnvVars, OTLPMetricsEndpointEnv))
		assert.False(t, isEnvVarSet(containerEnvVars, OTLPMetricsProtocolEnv))

		// verify logs exporter env vars are not added
		assert.False(t, isEnvVarSet(containerEnvVars, OTLPLogsEndpointEnv))
		assert.False(t, isEnvVarSet(containerEnvVars, OTLPLogsProtocolEnv))
	})
}

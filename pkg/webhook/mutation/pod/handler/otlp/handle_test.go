package otlp

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/otlp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/otlp/exporterconfig"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	webhookmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/webhook/mutation/pod/mutator"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testPodName       = "test-pod"
	testNamespaceName = "test-namespace"
	testDynakubeName  = "test-dynakube"
)

func TestHandler_Handle(t *testing.T) {
	t.Run("do not call mutators if OTLPExporterConfiguration is not defined", func(t *testing.T) {
		mockEnvVarMutator := webhookmock.NewMutator(t)
		mockResourceAttributeMutator := webhookmock.NewMutator(t)

		h := createTestHandler(mockEnvVarMutator, mockResourceAttributeMutator)

		dk := getTestDynakube()

		dk.Spec.OTLPExporterConfiguration = nil

		request := createTestMutationRequest(t, dk)

		err := h.Handle(request)
		require.NoError(t, err)
	})
	t.Run("call mutators if enabled", func(t *testing.T) {
		mockEnvVarMutator := webhookmock.NewMutator(t)
		mockResourceAttributeMutator := webhookmock.NewMutator(t)

		mockEnvVarMutator.On("IsEnabled", mock.Anything).Return(true)
		mockEnvVarMutator.On("IsInjected", mock.Anything).Return(false)

		mockEnvVarMutator.On("Mutate", mock.Anything).Return(nil)
		mockResourceAttributeMutator.On("Mutate", mock.Anything).Return(nil)

		h := createTestHandler(
			mockEnvVarMutator,
			mockResourceAttributeMutator,
			getTestTokenSecret(),
			getTestCertSecret(),
		)

		dk := getTestDynakube()
		request := createTestMutationRequest(t, dk)

		err := h.Handle(request)
		require.NoError(t, err)
	})
	t.Run("call mutators if no certificate secret is present, but ActiveGate is disabled", func(t *testing.T) {
		mockEnvVarMutator := webhookmock.NewMutator(t)
		mockResourceAttributeMutator := webhookmock.NewMutator(t)

		mockEnvVarMutator.On("IsEnabled", mock.Anything).Return(true)

		mockEnvVarMutator.On("IsInjected", mock.Anything).Return(false)

		mockEnvVarMutator.On("Mutate", mock.Anything).Return(nil)
		mockResourceAttributeMutator.On("Mutate", mock.Anything).Return(nil)

		h := createTestHandler(
			mockEnvVarMutator,
			mockResourceAttributeMutator,
			getTestTokenSecret(),
		)

		dk := getTestDynakube()

		dk.ActiveGate().TLSSecretName = ""
		dk.ActiveGate().Capabilities = []activegate.CapabilityDisplayName{}

		request := createTestMutationRequest(t, dk)

		err := h.Handle(request)
		require.NoError(t, err)
	})
	t.Run("call otlp exporter env var and resource attribute reinvocation if enabled", func(t *testing.T) {
		mockEnvVarMutator := webhookmock.NewMutator(t)
		mockResourceAttributeMutator := webhookmock.NewMutator(t)

		mockEnvVarMutator.On("IsEnabled", mock.Anything).Return(true)

		mockEnvVarMutator.On("IsInjected", mock.Anything).Return(true)

		mockEnvVarMutator.On("Reinvoke", mock.Anything).Return(true)
		mockResourceAttributeMutator.On("Reinvoke", mock.Anything).Return(true)

		h := createTestHandler(
			mockEnvVarMutator,
			mockResourceAttributeMutator,
			getTestTokenSecret(),
			getTestCertSecret(),
		)

		dk := getTestDynakube()
		request := createTestMutationRequest(t, dk)

		err := h.Handle(request)
		require.NoError(t, err)
	})
	t.Run("return error if exporter env var mutator returns error", func(t *testing.T) {
		mockEnvVarMutator := webhookmock.NewMutator(t)
		mockResourceAttributeMutator := webhookmock.NewMutator(t)

		mockEnvVarMutator.On("IsEnabled", mock.Anything).Return(true)
		mockEnvVarMutator.On("IsInjected", mock.Anything).Return(false)
		mockEnvVarMutator.On("Mutate", mock.Anything).Return(errors.New("error"))

		h := createTestHandler(
			mockEnvVarMutator,
			mockResourceAttributeMutator,
			getTestTokenSecret(),
			getTestCertSecret(),
		)

		dk := getTestDynakube()
		request := createTestMutationRequest(t, dk)

		err := h.Handle(request)
		require.Error(t, err)
	})
	t.Run("return error if resource attributes mutator returns an error", func(t *testing.T) {
		mockEnvVarMutator := webhookmock.NewMutator(t)
		mockResourceAttributeMutator := webhookmock.NewMutator(t)

		mockEnvVarMutator.On("IsEnabled", mock.Anything).Return(true)
		mockEnvVarMutator.On("IsInjected", mock.Anything).Return(false)
		mockEnvVarMutator.On("Mutate", mock.Anything).Return(nil)

		mockResourceAttributeMutator.On("Mutate", mock.Anything).Return(errors.New("error"))

		h := createTestHandler(
			mockEnvVarMutator,
			mockResourceAttributeMutator,
			getTestTokenSecret(),
			getTestCertSecret(),
		)

		dk := getTestDynakube()
		request := createTestMutationRequest(t, dk)

		err := h.Handle(request)
		assert.Error(t, err)
	})
	t.Run("skip injection and annotate when input secret missing and not replicable", func(t *testing.T) {
		mockEnvVarMutator := webhookmock.NewMutator(t)
		mockResourceAttributeMutator := webhookmock.NewMutator(t)

		// enable env var mutator so that secret presence is checked
		mockEnvVarMutator.On("IsEnabled", mock.Anything).Return(true)

		// create handler with NO secrets present
		h := createTestHandler(
			mockEnvVarMutator,
			mockResourceAttributeMutator,
		)

		dk := getTestDynakube()
		req := createTestMutationRequest(t, dk)

		err := h.Handle(req)
		require.NoError(t, err)

		// should be annotated as not injected due to missing input secret
		assert.Equal(t, "false", req.Pod.Annotations[mutator.AnnotationOTLPInjected])
		assert.Equal(t, NoOTLPExporterConfigSecretReason, req.Pod.Annotations[mutator.AnnotationOTLPReason])
	})
	t.Run("skip injection and annotate when certificate secret missing and not replicable", func(t *testing.T) {
		mockEnvVarMutator := webhookmock.NewMutator(t)
		mockResourceAttributeMutator := webhookmock.NewMutator(t)

		// enable env var mutator so that secret presence is checked
		mockEnvVarMutator.On("IsEnabled", mock.Anything).Return(true)

		// create handler with NO cert secret present
		h := createTestHandler(
			mockEnvVarMutator,
			mockResourceAttributeMutator,
			getTestTokenSecret(),
		)

		dk := getTestDynakube()
		req := createTestMutationRequest(t, dk)

		err := h.Handle(req)
		require.NoError(t, err)

		// should be annotated as not injected due to missing input secret
		assert.Equal(t, "false", req.Pod.Annotations[mutator.AnnotationOTLPInjected])
		assert.Equal(t, NoOTLPExporterActiveGateCertSecretReason, req.Pod.Annotations[mutator.AnnotationOTLPReason])

		// ensure mutators were not invoked
		mockEnvVarMutator.AssertNotCalled(t, "Mutate", mock.Anything)
		mockResourceAttributeMutator.AssertNotCalled(t, "IsEnabled", mock.Anything)
		mockResourceAttributeMutator.AssertNotCalled(t, "Mutate", mock.Anything)
	})
	t.Run("replicate input secret from source then proceed with injection", func(t *testing.T) {
		mockEnvVarMutator := webhookmock.NewMutator(t)
		mockResourceAttributeMutator := webhookmock.NewMutator(t)

		mockEnvVarMutator.On("IsEnabled", mock.Anything).Return(true)
		mockEnvVarMutator.On("IsInjected", mock.Anything).Return(false)
		mockEnvVarMutator.On("Mutate", mock.Anything).Return(nil)
		mockResourceAttributeMutator.On("Mutate", mock.Anything).Return(nil)

		dk := getTestDynakube()

		// provide only the SOURCE secret in the dynakube namespace; target secret absent
		sourceSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      exporterconfig.GetSourceConfigSecretName(dk.Name),
				Namespace: dk.Namespace,
			},
			Data: map[string][]byte{"token": []byte("abc")},
		}

		sourceCertSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      exporterconfig.GetSourceCertsSecretName(dk.Name),
				Namespace: dk.Namespace,
			},
			Data: map[string][]byte{"activegate-tls.cert": []byte("abc")},
		}

		h := createTestHandler(
			mockEnvVarMutator,
			mockResourceAttributeMutator,
			sourceSecret,
			sourceCertSecret,
		)

		req := createTestMutationRequest(t, dk)

		err := h.Handle(req)
		require.NoError(t, err)

		// target secret should now exist in the workload namespace
		var target corev1.Secret
		targetKey := client.ObjectKey{Name: consts.OTLPExporterSecretName, Namespace: testNamespaceName}
		require.NoError(t, h.apiReader.Get(req.Context, targetKey, &target))
		assert.Equal(t, consts.OTLPExporterSecretName, target.Name)
		assert.Equal(t, testNamespaceName, target.Namespace)
	})
}

func getTestTokenSecret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.OTLPExporterSecretName,
			Namespace: testNamespaceName,
		},
		Data: map[string][]byte{},
	}
}

func getTestCertSecret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.OTLPExporterCertsSecretName,
			Namespace: testNamespaceName,
		},
		Data: map[string][]byte{},
	}
}

func createTestHandler(envVarMutator, resourceAttributeMutator mutator.Mutator, objects ...client.Object) *Handler {
	fakeClient := fake.NewClient(objects...)

	return New(fakeClient, fakeClient, envVarMutator, resourceAttributeMutator)
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
					Traces:  &otlp.TracesSignal{},
					Metrics: &otlp.MetricsSignal{},
					Logs:    &otlp.LogsSignal{},
				},
			},
			ActiveGate: activegate.Spec{
				TLSSecretName: "tls-secret",
				Capabilities:  []activegate.CapabilityDisplayName{activegate.MetricsIngestCapability.DisplayName},
			},
		},
	}
}

func getTestDynakubeMeta() metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      testDynakubeName,
		Namespace: testNamespaceName,
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

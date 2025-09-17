package otlp

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	webhookmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/webhook/mutation/pod/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testPodName       = "test-pod"
	testNamespaceName = "test-namespace"
	testDynakubeName  = "test-dynakube"
)

func TestHandler_Handle(t *testing.T) {
	t.Run("auto-injection disabled via DynaKube annotations", func(t *testing.T) {
		mockEnvVarMutator := webhookmock.NewMutator(t)
		mockResourceAttributeMutator := webhookmock.NewMutator(t)

		h := createTestHandler(mockEnvVarMutator, mockResourceAttributeMutator)

		dk := getTestDynakube()

		dk.Annotations[exp.InjectionAutomaticKey] = "false"
		request := createTestMutationRequest(t, dk)

		err := h.Handle(request)
		assert.NoError(t, err)

		mockEnvVarMutator.AssertNotCalled(t, "Mutate")
		mockResourceAttributeMutator.AssertNotCalled(t, "Mutate")
	})
	t.Run("auto-injection disabled on pod", func(t *testing.T) {
		mockEnvVarMutator := webhookmock.NewMutator(t)
		mockResourceAttributeMutator := webhookmock.NewMutator(t)

		h := createTestHandler(mockEnvVarMutator, mockResourceAttributeMutator)

		dk := getTestDynakube()
		request := createTestMutationRequest(t, dk)

		request.Pod.Annotations[mutator.AnnotationDynatraceInject] = "false"

		err := h.Handle(request)
		assert.NoError(t, err)

		mockEnvVarMutator.AssertNotCalled(t, "Mutate")
		mockResourceAttributeMutator.AssertNotCalled(t, "Mutate")
	})
	t.Run("auto-injection enabled via 'feature.dynatrace.com/automatic-injection' default feature flag value", func(t *testing.T) {
		mockEnvVarMutator := webhookmock.NewMutator(t)
		mockResourceAttributeMutator := webhookmock.NewMutator(t)

		mockEnvVarMutator.On("IsEnabled", mock.Anything).Return(true)
		mockResourceAttributeMutator.On("IsEnabled", mock.Anything).Return(true)

		mockEnvVarMutator.On("Mutate", mock.Anything).Return(nil)
		mockResourceAttributeMutator.On("Mutate", mock.Anything).Return(nil)

		h := createTestHandler(mockEnvVarMutator, mockResourceAttributeMutator)

		dk := getTestDynakube()
		request := createTestMutationRequest(t, dk)

		err := h.Handle(request)
		assert.NoError(t, err)
	})
	t.Run("auto-injection enabled via 'otlp-exporter-configuration.dynatrace.com/inject' annotation on pod", func(t *testing.T) {
		mockEnvVarMutator := webhookmock.NewMutator(t)
		mockResourceAttributeMutator := webhookmock.NewMutator(t)

		mockEnvVarMutator.On("IsEnabled", mock.Anything).Return(true)
		mockResourceAttributeMutator.On("IsEnabled", mock.Anything).Return(true)

		mockEnvVarMutator.On("Mutate", mock.Anything).Return(nil)
		mockResourceAttributeMutator.On("Mutate", mock.Anything).Return(nil)

		h := createTestHandler(mockEnvVarMutator, mockResourceAttributeMutator)

		dk := getTestDynakube()
		request := createTestMutationRequest(t, dk)

		request.Pod.Annotations[AnnotationOTLPInjectionEnabled] = "true"

		err := h.Handle(request)
		assert.NoError(t, err)
	})
}

func createTestHandler(envVarMutator, resourceAttributeMutator mutator.Mutator) *Handler {
	return New(envVarMutator, resourceAttributeMutator)
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
			OneAgent: getCloudNativeSpec(),
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

func getCloudNativeSpec() oneagent.Spec {
	return oneagent.Spec{
		CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{
			AppInjectionSpec: oneagent.AppInjectionSpec{},
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

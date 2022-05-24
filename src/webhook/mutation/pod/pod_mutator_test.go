package pod

import (
	"context"
	"fmt"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/src/webhook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	testImage         = "test-image"
	testNamespaceName = "test-namespace"
	testClusterID     = "test-cluster-id"
	testPodName       = "test-pod"
	testDynakubeName  = "test-dynakube"
)

var testResourceRequirements = corev1.ResourceRequirements{
	Limits: map[corev1.ResourceName]resource.Quantity{
		corev1.ResourceCPU:    resource.MustParse("100m"),
		corev1.ResourceMemory: resource.MustParse("100Mi"),
	},
}

func TestHandle(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		mutator1 := createSimplePodMutatorMock()
		mutator2 := createSimplePodMutatorMock()
		dynakube := getTestDynakube()
		ctx := context.TODO()
		pod := getTestPod()
		namespace := getTestNamespace()
		request := createTestAdmissionRequest(pod)
		podWebhook := createTestWebhook(t, []dtwebhook.PodMutator{mutator1, mutator2}, []client.Object{dynakube, pod, namespace})

		response := podWebhook.Handle(ctx, *request)
		require.NotNil(t, response)
		assert.True(t, response.Allowed)
		assert.Nil(t, response.Result)
		assert.NotNil(t, response.Patches)
		mutator1.(*dtwebhook.PodMutatorMock).AssertNumberOfCalls(t, "Enabled", 1)
		mutator1.(*dtwebhook.PodMutatorMock).AssertNumberOfCalls(t, "Mutate", 1)
		mutator2.(*dtwebhook.PodMutatorMock).AssertNumberOfCalls(t, "Enabled", 1)
		mutator2.(*dtwebhook.PodMutatorMock).AssertNumberOfCalls(t, "Mutate", 1)
	})
	t.Run("sad path", func(t *testing.T) {
		sadMutator := createFailPodMutatorMock()
		dynakube := getTestDynakube()
		ctx := context.TODO()
		pod := getTestPod()
		namespace := getTestNamespace()
		request := createTestAdmissionRequest(pod)
		podWebhook := createTestWebhook(t, []dtwebhook.PodMutator{sadMutator}, []client.Object{dynakube, pod, namespace})

		response := podWebhook.Handle(ctx, *request)
		require.NotNil(t, response)
		assert.True(t, response.Allowed)
		assert.Contains(t, response.Result.Message, "Failed")
		assert.Nil(t, response.Patches)
		sadMutator.(*dtwebhook.PodMutatorMock).AssertNumberOfCalls(t, "Enabled", 1)
		sadMutator.(*dtwebhook.PodMutatorMock).AssertNumberOfCalls(t, "Mutate", 1)
	})
}

func TestHandlePodMutation(t *testing.T) {
	t.Run("should call both mutators, initContainer and annotation added, no error", func(t *testing.T) {
		mutator1 := createSimplePodMutatorMock()
		mutator2 := createSimplePodMutatorMock()
		dynakube := getTestDynakube()
		podWebhook := createTestWebhook(t, []dtwebhook.PodMutator{mutator1, mutator2}, nil)
		mutationRequest := createTestMutationRequest(dynakube)

		err := podWebhook.handlePodMutation(mutationRequest)
		require.NoError(t, err)
		assert.NotNil(t, mutationRequest.InitContainer)
		assert.Len(t, mutationRequest.Pod.Spec.InitContainers, 2)
		assert.Equal(t, mutationRequest.Pod.Spec.InitContainers[1].SecurityContext, testSecurityContext)
		assert.Equal(t, mutationRequest.Pod.Spec.InitContainers[1].Resources, testResourceRequirements)
		assert.Equal(t, "true", mutationRequest.Pod.Annotations[dtwebhook.AnnotationDynatraceInjected])
		mutator1.(*dtwebhook.PodMutatorMock).AssertCalled(t, "Enabled", mutationRequest.Pod)
		mutator1.(*dtwebhook.PodMutatorMock).AssertCalled(t, "Mutate", mutationRequest)
		mutator2.(*dtwebhook.PodMutatorMock).AssertCalled(t, "Enabled", mutationRequest.Pod)
		mutator2.(*dtwebhook.PodMutatorMock).AssertCalled(t, "Mutate", mutationRequest)
	})
	t.Run("should call 1 mutator, 1 error, no initContainer and annotation", func(t *testing.T) {
		sadMutator := createFailPodMutatorMock()
		happyMutator := createSimplePodMutatorMock()
		dynakube := getTestDynakube()
		podWebhook := createTestWebhook(t, []dtwebhook.PodMutator{sadMutator, happyMutator}, nil)
		mutationRequest := createTestMutationRequest(dynakube)

		err := podWebhook.handlePodMutation(mutationRequest)
		require.Error(t, err)
		assert.NotNil(t, mutationRequest.InitContainer)
		assert.Len(t, mutationRequest.Pod.Spec.InitContainers, 1)
		assert.NotEqual(t, "true", mutationRequest.Pod.Annotations[dtwebhook.AnnotationDynatraceInjected])
		sadMutator.(*dtwebhook.PodMutatorMock).AssertCalled(t, "Enabled", mutationRequest.Pod)
		sadMutator.(*dtwebhook.PodMutatorMock).AssertCalled(t, "Mutate", mutationRequest)
		happyMutator.(*dtwebhook.PodMutatorMock).AssertNotCalled(t, "Enabled", mock.Anything)
		happyMutator.(*dtwebhook.PodMutatorMock).AssertNotCalled(t, "Mutate", mock.Anything)
	})
}

func TestHandlePodReinvocation(t *testing.T) {
	t.Run("no reinvocation feature-flag, no update", func(t *testing.T) {
		mutator1 := createAlreadyInjectedPodMutatorMock()
		mutator2 := createAlreadyInjectedPodMutatorMock()
		dynakube := getTestDynakube()
		podWebhook := createTestWebhook(t, []dtwebhook.PodMutator{mutator1, mutator2}, nil)
		mutationRequest := createTestMutationRequest(dynakube)

		updated := podWebhook.handlePodReinvocation(mutationRequest)
		require.False(t, updated)
		mutator1.(*dtwebhook.PodMutatorMock).AssertNotCalled(t, "Enabled", mutationRequest.Pod)
		mutator1.(*dtwebhook.PodMutatorMock).AssertNotCalled(t, "Reinvoke", mutationRequest.ToReinvocationRequest())
		mutator2.(*dtwebhook.PodMutatorMock).AssertNotCalled(t, "Enabled", mutationRequest.Pod)
		mutator2.(*dtwebhook.PodMutatorMock).AssertNotCalled(t, "Reinvoke", mutationRequest.ToReinvocationRequest())
	})
	t.Run("should call both mutators, updated == true", func(t *testing.T) {
		mutator1 := createAlreadyInjectedPodMutatorMock()
		mutator2 := createAlreadyInjectedPodMutatorMock()
		dynakube := getTestDynakube()
		dynakube.Annotations = map[string]string{dynatracev1beta1.AnnotationFeatureEnableWebhookReinvocationPolicy: "true"}
		podWebhook := createTestWebhook(t, []dtwebhook.PodMutator{mutator1, mutator2}, nil)
		mutationRequest := createTestMutationRequest(dynakube)

		updated := podWebhook.handlePodReinvocation(mutationRequest)
		require.True(t, updated)
		mutator1.(*dtwebhook.PodMutatorMock).AssertCalled(t, "Enabled", mutationRequest.Pod)
		mutator1.(*dtwebhook.PodMutatorMock).AssertCalled(t, "Reinvoke", mutationRequest.ToReinvocationRequest())
		mutator2.(*dtwebhook.PodMutatorMock).AssertCalled(t, "Enabled", mutationRequest.Pod)
		mutator2.(*dtwebhook.PodMutatorMock).AssertCalled(t, "Reinvoke", mutationRequest.ToReinvocationRequest())
	})
	t.Run("should call both mutator, only 1 update, updated == true", func(t *testing.T) {
		sadMutator := createFailPodMutatorMock()
		happyMutator := createAlreadyInjectedPodMutatorMock()
		dynakube := getTestDynakube()
		dynakube.Annotations = map[string]string{dynatracev1beta1.AnnotationFeatureEnableWebhookReinvocationPolicy: "true"}
		podWebhook := createTestWebhook(t, []dtwebhook.PodMutator{sadMutator, happyMutator}, nil)
		mutationRequest := createTestMutationRequest(dynakube)

		updated := podWebhook.handlePodReinvocation(mutationRequest)
		require.True(t, updated)
		sadMutator.(*dtwebhook.PodMutatorMock).AssertCalled(t, "Enabled", mutationRequest.Pod)
		sadMutator.(*dtwebhook.PodMutatorMock).AssertCalled(t, "Reinvoke", mutationRequest.ToReinvocationRequest())
		happyMutator.(*dtwebhook.PodMutatorMock).AssertCalled(t, "Enabled", mutationRequest.Pod)
		happyMutator.(*dtwebhook.PodMutatorMock).AssertCalled(t, "Reinvoke", mutationRequest.ToReinvocationRequest())
	})
	t.Run("should call mutator, no update", func(t *testing.T) {
		sadMutator := createFailPodMutatorMock()
		dynakube := getTestDynakube()
		dynakube.Annotations = map[string]string{dynatracev1beta1.AnnotationFeatureEnableWebhookReinvocationPolicy: "true"}
		podWebhook := createTestWebhook(t, []dtwebhook.PodMutator{sadMutator}, nil)
		mutationRequest := createTestMutationRequest(dynakube)

		updated := podWebhook.handlePodReinvocation(mutationRequest)
		require.False(t, updated)
		sadMutator.(*dtwebhook.PodMutatorMock).AssertCalled(t, "Enabled", mutationRequest.Pod)
		sadMutator.(*dtwebhook.PodMutatorMock).AssertCalled(t, "Reinvoke", mutationRequest.ToReinvocationRequest())
	})
}

func createTestWebhook(t *testing.T, mutators []dtwebhook.PodMutator, objects []client.Object) *podMutatorWebhook {
	decoder, err := admission.NewDecoder(scheme.Scheme)
	require.NoError(t, err)
	return &podMutatorWebhook{
		apiReader:        fake.NewClient(objects...),
		decoder:          decoder,
		recorder:         podMutatorEventRecorder{recorder: record.NewFakeRecorder(10), pod: &corev1.Pod{}, dynakube: getTestDynakube()},
		webhookImage:     testImage,
		webhookNamespace: testNamespaceName,
		clusterID:        testClusterID,
		apmExists:        false,
		mutators:         mutators,
	}
}

func createSimplePodMutatorMock() dtwebhook.PodMutator {
	mutator := dtwebhook.PodMutatorMock{}
	mutator.On("Enabled", mock.Anything).Return(true)
	mutator.On("Injected", mock.Anything).Return(false)
	mutator.On("Mutate", mock.Anything).Return(nil)
	mutator.On("Reinvoke", mock.Anything).Return(true)
	return &mutator
}

func createAlreadyInjectedPodMutatorMock() dtwebhook.PodMutator {
	mutator := dtwebhook.PodMutatorMock{}
	mutator.On("Enabled", mock.Anything).Return(true)
	mutator.On("Injected", mock.Anything).Return(true)
	mutator.On("Mutate", mock.Anything).Return(nil)
	mutator.On("Reinvoke", mock.Anything).Return(true)
	return &mutator
}

func createFailPodMutatorMock() dtwebhook.PodMutator {
	mutator := dtwebhook.PodMutatorMock{}
	mutator.On("Enabled", mock.Anything).Return(true)
	mutator.On("Injected", mock.Anything).Return(false)
	mutator.On("Mutate", mock.Anything).Return(fmt.Errorf("BOOM"))
	mutator.On("Reinvoke", mock.Anything).Return(false)
	return &mutator
}

func getTestDynakube() *dynatracev1beta1.DynaKube {
	return &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testDynakubeName,
			Namespace: testNamespaceName,
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			OneAgent: dynatracev1beta1.OneAgentSpec{
				CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{
					AppInjectionSpec: dynatracev1beta1.AppInjectionSpec{
						InitResources: testResourceRequirements,
					},
				},
			},
		},
	}
}

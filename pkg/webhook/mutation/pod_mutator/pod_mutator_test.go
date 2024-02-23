package pod_mutator

import (
	"context"
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	webhookmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/webhook"
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

type mutatorTest struct {
	name           string
	mutators       []dtwebhook.PodMutator
	testPod        *corev1.Pod
	objects        []client.Object
	expectedResult func(t *testing.T, response *admission.Response, mutators []dtwebhook.PodMutator)
}

func TestMutator(t *testing.T) {
	tests := []mutatorTest{
		{
			name:     "happy path",
			mutators: []dtwebhook.PodMutator{createSimplePodMutatorMock(t), createSimplePodMutatorMock(t)},
			testPod:  getTestPod(),
			objects:  []client.Object{getTestDynakube(), getTestNamespace()},
			expectedResult: func(t *testing.T, response *admission.Response, mutators []dtwebhook.PodMutator) {
				require.NotNil(t, response)
				assert.True(t, response.Allowed)
				assert.Nil(t, response.Result)
				assert.NotNil(t, response.Patches)

				for _, mutator := range mutators {
					assertPodMutatorCalls(t, mutator, 1)
				}
			},
		},
		{
			name:     "disable all mutators with dynatrace.com/inject",
			mutators: []dtwebhook.PodMutator{createSimplePodMutatorMock(t), createSimplePodMutatorMock(t)},
			testPod:  getTestPodWithInjectionDisabled(),
			objects:  []client.Object{getTestDynakube(), getTestNamespace()},
			expectedResult: func(t *testing.T, response *admission.Response, mutators []dtwebhook.PodMutator) {
				require.NotNil(t, response)
				assert.True(t, response.Allowed)
				assert.NotNil(t, response.Result)
				assert.Nil(t, response.Patches)

				for _, mutator := range mutators {
					assertPodMutatorCalls(t, mutator, 0)
				}
			},
		},
		{
			name:     "sad path",
			mutators: []dtwebhook.PodMutator{createFailPodMutatorMock(t)},
			testPod:  getTestPod(),
			objects:  []client.Object{getTestDynakube(), getTestNamespace()},
			expectedResult: func(t *testing.T, response *admission.Response, mutators []dtwebhook.PodMutator) {
				require.NotNil(t, response)
				assert.True(t, response.Allowed)
				assert.Contains(t, response.Result.Message, "Failed")
				assert.Nil(t, response.Patches)

				for _, mutator := range mutators {
					assertPodMutatorCalls(t, mutator, 1)
				}

				// Logging newline so go test can parse the output correctly
				log.Info("")
			},
		},
		{
			name:     "oc debug pod",
			mutators: []dtwebhook.PodMutator{createSimplePodMutatorMock(t)},
			testPod:  getTestPodWithOcDebugPodAnnotations(),
			objects:  []client.Object{getTestDynakube(), getTestNamespace()},
			expectedResult: func(t *testing.T, response *admission.Response, mutators []dtwebhook.PodMutator) {
				require.NotNil(t, response)
				assert.True(t, response.Allowed)
				assert.NotNil(t, response.Result)
				assert.Nil(t, response.Patches)
				assert.Nil(t, response.Patch)

				for _, mutator := range mutators {
					assertPodMutatorCalls(t, mutator, 0)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			request := createTestAdmissionRequest(test.testPod)
			// merge test objects with the test pod
			objects := test.objects
			objects = append(objects, test.testPod)
			podWebhook := createTestWebhook(test.mutators, objects)

			response := podWebhook.Handle(ctx, *request)
			test.expectedResult(t, &response, test.mutators)
		})
	}
}

func TestHandlePodMutation(t *testing.T) {
	t.Run("should call both mutators, initContainer and annotation added, no error", func(t *testing.T) {
		mutator1 := createSimplePodMutatorMock(t)
		mutator2 := createSimplePodMutatorMock(t)
		dynakube := getTestDynakube()
		podWebhook := createTestWebhook([]dtwebhook.PodMutator{mutator1, mutator2}, nil)
		mutationRequest := createTestMutationRequest(dynakube)

		err := podWebhook.handlePodMutation(context.Background(), mutationRequest)
		require.NoError(t, err)
		assert.NotNil(t, mutationRequest.InstallContainer)
		assert.Len(t, mutationRequest.Pod.Spec.InitContainers, 2)

		initSecurityContext := mutationRequest.Pod.Spec.InitContainers[1].SecurityContext
		require.NotNil(t, initSecurityContext)

		require.NotNil(t, initSecurityContext.Privileged)
		assert.False(t, *initSecurityContext.Privileged)

		require.NotNil(t, initSecurityContext.AllowPrivilegeEscalation)
		assert.False(t, *initSecurityContext.AllowPrivilegeEscalation)

		require.NotNil(t, initSecurityContext.ReadOnlyRootFilesystem)
		assert.True(t, *initSecurityContext.ReadOnlyRootFilesystem)

		assert.NotNil(t, initSecurityContext.RunAsNonRoot)
		assert.True(t, *initSecurityContext.RunAsNonRoot)

		assert.Equal(t, mutationRequest.Pod.Spec.InitContainers[1].Resources, testResourceRequirements)
		assert.Equal(t, "true", mutationRequest.Pod.Annotations[dtwebhook.AnnotationDynatraceInjected])
		mutator1.AssertCalled(t, "Enabled", mutationRequest.BaseRequest)
		mutator1.AssertCalled(t, "Mutate", mock.Anything, mutationRequest)
		mutator2.AssertCalled(t, "Enabled", mutationRequest.BaseRequest)
		mutator2.AssertCalled(t, "Mutate", mock.Anything, mutationRequest)
	})
	t.Run("should call 1 mutator, 1 error, no initContainer and annotation", func(t *testing.T) {
		sadMutator := createFailPodMutatorMock(t)
		happyMutator := createSimplePodMutatorMock(t)
		dynakube := getTestDynakube()
		podWebhook := createTestWebhook([]dtwebhook.PodMutator{sadMutator, happyMutator}, nil)
		mutationRequest := createTestMutationRequest(dynakube)

		err := podWebhook.handlePodMutation(context.Background(), mutationRequest)
		require.Error(t, err)
		assert.NotNil(t, mutationRequest.InstallContainer)
		assert.Len(t, mutationRequest.Pod.Spec.InitContainers, 1)
		assert.NotEqual(t, "true", mutationRequest.Pod.Annotations[dtwebhook.AnnotationDynatraceInjected])
		sadMutator.AssertCalled(t, "Enabled", mutationRequest.BaseRequest)
		sadMutator.AssertCalled(t, "Mutate", mock.Anything, mutationRequest)
		happyMutator.AssertNotCalled(t, "Enabled", mock.Anything)
		happyMutator.AssertNotCalled(t, "Mutate", mock.Anything, mock.Anything)
	})
}

func TestHandlePodReinvocation(t *testing.T) {
	t.Run("should call both mutators, updated == true", func(t *testing.T) {
		mutator1 := createAlreadyInjectedPodMutatorMock(t)
		mutator2 := createAlreadyInjectedPodMutatorMock(t)
		dynakube := getTestDynakube()
		podWebhook := createTestWebhook([]dtwebhook.PodMutator{mutator1, mutator2}, nil)
		mutationRequest := createTestMutationRequest(dynakube)

		updated := podWebhook.handlePodReinvocation(context.Background(), mutationRequest)
		require.True(t, updated)
		mutator1.AssertCalled(t, "Enabled", mutationRequest.BaseRequest)
		mutator1.AssertCalled(t, "Reinvoke", mutationRequest.ToReinvocationRequest())
		mutator2.AssertCalled(t, "Enabled", mutationRequest.BaseRequest)
		mutator2.AssertCalled(t, "Reinvoke", mutationRequest.ToReinvocationRequest())
	})
	t.Run("should call both mutator, only 1 update, updated == true", func(t *testing.T) {
		failingMutator := createFailPodMutatorMock(t)
		workingMutator := createAlreadyInjectedPodMutatorMock(t)
		dynakube := getTestDynakube()
		podWebhook := createTestWebhook([]dtwebhook.PodMutator{failingMutator, workingMutator}, nil)
		mutationRequest := createTestMutationRequest(dynakube)

		updated := podWebhook.handlePodReinvocation(context.Background(), mutationRequest)
		require.True(t, updated)
		failingMutator.AssertCalled(t, "Enabled", mutationRequest.BaseRequest)
		failingMutator.AssertCalled(t, "Reinvoke", mutationRequest.ToReinvocationRequest())
		workingMutator.AssertCalled(t, "Enabled", mutationRequest.BaseRequest)
		workingMutator.AssertCalled(t, "Reinvoke", mutationRequest.ToReinvocationRequest())
	})
	t.Run("should call mutator, no update", func(t *testing.T) {
		failingMutator := createFailPodMutatorMock(t)
		dynakube := getTestDynakube()
		podWebhook := createTestWebhook([]dtwebhook.PodMutator{failingMutator}, nil)
		mutationRequest := createTestMutationRequest(dynakube)

		updated := podWebhook.handlePodReinvocation(context.Background(), mutationRequest)
		require.False(t, updated)
		failingMutator.AssertCalled(t, "Enabled", mutationRequest.BaseRequest)
		failingMutator.AssertCalled(t, "Reinvoke", mutationRequest.ToReinvocationRequest())
		failingMutator.AssertNotCalled(t, "Injected", mock.Anything)
		failingMutator.AssertNotCalled(t, "Mutated", mock.Anything, mock.Anything)
	})
}

func assertPodMutatorCalls(t *testing.T, mutator dtwebhook.PodMutator, expectedCalls int) {
	mock, ok := mutator.(*webhookmock.PodMutator)
	if !ok {
		t.Fatalf("assertPodMutatorCalls: mutator is not a mock")
	}

	mock.AssertNumberOfCalls(t, "Enabled", expectedCalls)
	mock.AssertNumberOfCalls(t, "Mutate", expectedCalls)
}

func getTestPodWithInjectionDisabled() *corev1.Pod {
	pod := getTestPod()
	pod.Annotations = map[string]string{
		dtwebhook.AnnotationDynatraceInject: "false",
	}

	return pod
}

func getTestPodWithOcDebugPodAnnotations() *corev1.Pod {
	pod := getTestPod()
	pod.Annotations = map[string]string{
		ocDebugAnnotationsContainer: "true",
		ocDebugAnnotationsResource:  "true",
	}

	return pod
}

func createTestWebhook(mutators []dtwebhook.PodMutator, objects []client.Object) *podMutatorWebhook {
	decoder := admission.NewDecoder(scheme.Scheme)

	return &podMutatorWebhook{
		apiReader:        fake.NewClient(objects...),
		decoder:          *decoder,
		recorder:         podMutatorEventRecorder{recorder: record.NewFakeRecorder(10), pod: &corev1.Pod{}, dynakube: getTestDynakube()},
		webhookImage:     testImage,
		webhookNamespace: testNamespaceName,
		clusterID:        testClusterID,
		apmExists:        false,
		mutators:         mutators,
	}
}

func createSimplePodMutatorMock(t *testing.T) *webhookmock.PodMutator {
	mutator := webhookmock.NewPodMutator(t)
	mutator.On("Enabled", mock.Anything).Return(true).Maybe()
	mutator.On("Injected", mock.Anything).Return(false).Maybe()
	mutator.On("Mutate", mock.Anything, mock.Anything).Return(nil).Maybe()
	mutator.On("Reinvoke", mock.Anything).Return(true).Maybe()

	return mutator
}

func createAlreadyInjectedPodMutatorMock(t *testing.T) *webhookmock.PodMutator {
	mutator := webhookmock.NewPodMutator(t)
	mutator.On("Enabled", mock.Anything).Return(true).Maybe()
	mutator.On("Injected", mock.Anything).Return(true).Maybe()
	mutator.On("Mutate", mock.Anything, mock.Anything).Return(nil).Maybe()
	mutator.On("Reinvoke", mock.Anything).Return(true).Maybe()

	return mutator
}

func createFailPodMutatorMock(t *testing.T) *webhookmock.PodMutator {
	mutator := webhookmock.NewPodMutator(t)
	mutator.On("Enabled", mock.Anything).Return(true).Maybe()
	mutator.On("Injected", mock.Anything).Return(false).Maybe()
	mutator.On("Mutate", mock.Anything, mock.Anything).Return(fmt.Errorf("BOOM")).Maybe()
	mutator.On("Reinvoke", mock.Anything).Return(false).Maybe()

	return mutator
}

func getTestDynakube() *dynatracev1beta1.DynaKube {
	return &dynatracev1beta1.DynaKube{
		ObjectMeta: getTestDynakubeMeta(),
		Spec: dynatracev1beta1.DynaKubeSpec{
			OneAgent: getCloudNativeSpec(&testResourceRequirements),
		},
	}
}

func getTestDynakubeNoInitLimits() *dynatracev1beta1.DynaKube {
	return &dynatracev1beta1.DynaKube{
		ObjectMeta: getTestDynakubeMeta(),
		Spec: dynatracev1beta1.DynaKubeSpec{
			OneAgent: getCloudNativeSpec(nil),
		},
	}
}

func getTestDynakubeDefaultAppMon() *dynatracev1beta1.DynaKube {
	return &dynatracev1beta1.DynaKube{
		ObjectMeta: getTestDynakubeMeta(),
		Spec: dynatracev1beta1.DynaKubeSpec{
			OneAgent: dynatracev1beta1.OneAgentSpec{
				ApplicationMonitoring: &dynatracev1beta1.ApplicationMonitoringSpec{},
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

func getCloudNativeSpec(initResources *corev1.ResourceRequirements) dynatracev1beta1.OneAgentSpec {
	return dynatracev1beta1.OneAgentSpec{
		CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{
			AppInjectionSpec: dynatracev1beta1.AppInjectionSpec{
				InitResources: initResources,
			},
		},
	}
}

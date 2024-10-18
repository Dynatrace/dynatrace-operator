package pod

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/startup"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/metadata"
	oamutation "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/oneagent"
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

// TestDoubleInjection is special test case for making sure that we do not inject the init-container 2 times incase 1 of the mutators are skipped.
// The mutators are intentionally NOT mocked, as to mock them properly for this scenario you would need to basically reimplement them in the mock.
// This test is necessary as the current interface is not ready to handle the scenario properly.
// Scenario: OneAgent mutation is Enabled however needs to be skipped due to not meeting the requirements, so it needs to annotate but not fully inject
func TestDoubleInjection(t *testing.T) {
	noCommunicationHostDK := getTestDynakube()
	fakeClient := fake.NewClient(noCommunicationHostDK, getTestNamespace())
	podWebhook := &webhook{
		apiReader:        fakeClient,
		decoder:          admission.NewDecoder(scheme.Scheme),
		recorder:         eventRecorder{recorder: record.NewFakeRecorder(10), pod: &corev1.Pod{}, dk: noCommunicationHostDK},
		webhookImage:     testImage,
		webhookNamespace: testNamespaceName,
		clusterID:        testClusterID,
		apmExists:        false,
		mutators: []dtwebhook.PodMutator{
			oamutation.NewMutator(
				testImage,
				testClusterID,
				testNamespaceName,
				fakeClient,
				fakeClient,
			),
			metadata.NewMutator(
				testNamespaceName,
				fakeClient,
				fakeClient,
				fakeClient,
			),
		},
	}

	pod := getTestPod()

	request := createTestAdmissionRequest(pod)

	response := podWebhook.Handle(context.Background(), *request)
	require.NotNil(t, response)
	assert.True(t, response.Allowed)
	assert.Nil(t, response.Result)
	require.Len(t, response.Patches, 2)

	allowedPatchPaths := []string{
		"/spec/initContainers/1",
		"/metadata/annotations",
	}
	alreadySeenPaths := []string{}

	for _, patch := range response.Patches {
		path := patch.Path
		assert.NotContains(t, alreadySeenPaths, path)
		assert.Contains(t, allowedPatchPaths, path)
		alreadySeenPaths = append(alreadySeenPaths, path)
	}

	// simulate initial mutation, annotations + init-container <== skip in case on communication hosts
	pod.Annotations = map[string]string{
		dtwebhook.AnnotationOneAgentInjected: "false",
		dtwebhook.AnnotationOneAgentReason:   oamutation.EmptyConnectionInfoReason,
	}
	pod.Spec.InitContainers = append(pod.Spec.InitContainers, corev1.Container{Name: dtwebhook.InstallContainerName})

	// adding communicationHost to the dynakube to make the scenario more complicated
	// it shouldn't try to mutate the pod because now it could be enabled, that is just asking for trouble.
	communicationHostDK := getTestDynakube()
	communicationHostDK.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts = []dynakube.CommunicationHostStatus{{Host: "test"}}
	fakeClient = fake.NewClient(communicationHostDK, getTestNamespace())
	podWebhook.apiReader = fakeClient

	// simulate a Reinvocation
	request = createTestAdmissionRequest(pod)
	response = podWebhook.Handle(context.Background(), *request)

	require.NotNil(t, response)
	assert.Equal(t, admission.Patched(""), response)
}

func TestHandlePodMutation(t *testing.T) {
	t.Run("should call both mutators, initContainer and annotation added, no error", func(t *testing.T) {
		mutator1 := createSimplePodMutatorMock(t)
		mutator2 := createSimplePodMutatorMock(t)
		dk := getTestDynakube()
		podWebhook := createTestWebhook([]dtwebhook.PodMutator{mutator1, mutator2}, nil)
		mutationRequest := createTestMutationRequest(dk)

		err := podWebhook.handlePodMutation(context.Background(), mutationRequest)
		require.NoError(t, err)
		assert.NotNil(t, mutationRequest.InstallContainer)

		require.Len(t, mutationRequest.Pod.Spec.InitContainers, 2)

		assertContainersInfo(t, mutationRequest.ToReinvocationRequest(), &mutationRequest.Pod.Spec.InitContainers[1])

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
	t.Run("should call 1 webhook, 1 error, no initContainer and annotation", func(t *testing.T) {
		sadMutator := createFailPodMutatorMock(t)
		happyMutator := createSimplePodMutatorMock(t)
		dk := getTestDynakube()
		podWebhook := createTestWebhook([]dtwebhook.PodMutator{sadMutator, happyMutator}, nil)
		mutationRequest := createTestMutationRequest(dk)

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
		dk := getTestDynakube()
		podWebhook := createTestWebhook([]dtwebhook.PodMutator{mutator1, mutator2}, nil)
		mutationRequest := createTestMutationRequestWithInjectedPod(dk)

		updated := podWebhook.handlePodReinvocation(mutationRequest)
		require.True(t, updated)

		require.Len(t, mutationRequest.Pod.Spec.InitContainers, 2)
		assertContainersInfo(t, mutationRequest.ToReinvocationRequest(), &mutationRequest.Pod.Spec.InitContainers[1])

		mutator1.AssertCalled(t, "Enabled", mutationRequest.BaseRequest)
		mutator1.AssertCalled(t, "Reinvoke", mutationRequest.ToReinvocationRequest())
		mutator2.AssertCalled(t, "Enabled", mutationRequest.BaseRequest)
		mutator2.AssertCalled(t, "Reinvoke", mutationRequest.ToReinvocationRequest())
	})
	t.Run("should call both webhook, only 1 update, updated == true", func(t *testing.T) {
		failingMutator := createFailPodMutatorMock(t)
		workingMutator := createAlreadyInjectedPodMutatorMock(t)
		dk := getTestDynakube()
		podWebhook := createTestWebhook([]dtwebhook.PodMutator{failingMutator, workingMutator}, nil)
		mutationRequest := createTestMutationRequestWithInjectedPod(dk)

		updated := podWebhook.handlePodReinvocation(mutationRequest)
		require.True(t, updated)
		failingMutator.AssertCalled(t, "Enabled", mutationRequest.BaseRequest)
		failingMutator.AssertCalled(t, "Reinvoke", mutationRequest.ToReinvocationRequest())
		workingMutator.AssertCalled(t, "Enabled", mutationRequest.BaseRequest)
		workingMutator.AssertCalled(t, "Reinvoke", mutationRequest.ToReinvocationRequest())
	})
	t.Run("should call webhook, no update", func(t *testing.T) {
		failingMutator := createFailPodMutatorMock(t)
		dk := getTestDynakube()
		podWebhook := createTestWebhook([]dtwebhook.PodMutator{failingMutator}, nil)
		mutationRequest := createTestMutationRequestWithInjectedPod(dk)

		updated := podWebhook.handlePodReinvocation(mutationRequest)
		require.False(t, updated)
		failingMutator.AssertCalled(t, "Enabled", mutationRequest.BaseRequest)
		failingMutator.AssertCalled(t, "Reinvoke", mutationRequest.ToReinvocationRequest())
		failingMutator.AssertNotCalled(t, "Injected", mock.Anything)
		failingMutator.AssertNotCalled(t, "Mutated", mock.Anything, mock.Anything)
	})
}

func assertContainersInfo(t *testing.T, request *dtwebhook.ReinvocationRequest, installContainer *corev1.Container) {
	rawContainerInfo := env.FindEnvVar(installContainer.Env, consts.ContainerInfoEnv)
	require.NotNil(t, rawContainerInfo)

	var containerInfo []startup.ContainerInfo
	err := json.Unmarshal([]byte(rawContainerInfo.Value), &containerInfo)
	require.NoError(t, err)

	for _, container := range request.Pod.Spec.Containers {
		found := false

		for _, info := range containerInfo {
			if container.Name == info.Name {
				assert.Equal(t, container.Image, info.Image)

				found = true

				break
			}
		}

		require.True(t, found)
	}
}

func assertPodMutatorCalls(t *testing.T, mutator dtwebhook.PodMutator, expectedCalls int) {
	mock, ok := mutator.(*webhookmock.PodMutator)
	if !ok {
		t.Fatalf("assertPodMutatorCalls: webhook is not a mock")
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

func createTestWebhook(mutators []dtwebhook.PodMutator, objects []client.Object) *webhook {
	decoder := admission.NewDecoder(scheme.Scheme)

	return &webhook{
		apiReader:        fake.NewClient(objects...),
		decoder:          decoder,
		recorder:         eventRecorder{recorder: record.NewFakeRecorder(10), pod: &corev1.Pod{}, dk: getTestDynakube()},
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

func getTestDynakube() *dynakube.DynaKube {
	return &dynakube.DynaKube{
		ObjectMeta: getTestDynakubeMeta(),
		Spec: dynakube.DynaKubeSpec{
			OneAgent: getCloudNativeSpec(&testResourceRequirements),
		},
	}
}

func getTestDynakubeNoInitLimits() *dynakube.DynaKube {
	return &dynakube.DynaKube{
		ObjectMeta: getTestDynakubeMeta(),
		Spec: dynakube.DynaKubeSpec{
			OneAgent: getCloudNativeSpec(nil),
		},
	}
}

func getTestDynakubeDefaultAppMon() *dynakube.DynaKube {
	return &dynakube.DynaKube{
		ObjectMeta: getTestDynakubeMeta(),
		Spec: dynakube.DynaKubeSpec{
			OneAgent: dynakube.OneAgentSpec{
				ApplicationMonitoring: &dynakube.ApplicationMonitoringSpec{},
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

func getCloudNativeSpec(initResources *corev1.ResourceRequirements) dynakube.OneAgentSpec {
	return dynakube.OneAgentSpec{
		CloudNativeFullStack: &dynakube.CloudNativeFullStackSpec{
			AppInjectionSpec: dynakube.AppInjectionSpec{
				InitResources: initResources,
			},
		},
	}
}

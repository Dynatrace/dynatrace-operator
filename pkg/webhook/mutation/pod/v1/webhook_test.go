package v1

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/startup"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/common/events"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/v1/metadata"
	oamutation "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/v1/oneagent"
	webhookmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/webhook"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
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

func TestInjector(t *testing.T) {
	t.Run("not yet injected => mutate", func(t *testing.T) {
		ctx := context.Background()

		injector := createTestInjector([]dtwebhook.PodMutator{createSimplePodMutatorMock(t), createSimplePodMutatorMock(t)}, false)
		request := createTestMutationRequest(getTestDynakube())

		err := injector.Handle(ctx, request)
		require.NoError(t, err)

		for _, mutator := range injector.mutators {
			assertMutateCalls(t, mutator, 1)
		}
	})

	t.Run("fail => error", func(t *testing.T) {
		ctx := context.Background()

		injector := createTestInjector([]dtwebhook.PodMutator{createFailPodMutatorMock(t)}, false)
		request := createTestMutationRequest(getTestDynakube())

		err := injector.Handle(ctx, request)
		require.Error(t, err)

		for _, mutator := range injector.mutators {
			assertMutateCalls(t, mutator, 1)
		}
	})

	t.Run("already injected => reinvoke", func(t *testing.T) {
		ctx := context.Background()

		injector := createTestInjector([]dtwebhook.PodMutator{createAlreadyInjectedPodMutatorMock(t), createAlreadyInjectedPodMutatorMock(t)}, true)
		request := createTestMutationRequestWithInjectedPod(getTestDynakube())

		err := injector.Handle(ctx, request)
		require.NoError(t, err)

		for _, mutator := range injector.mutators {
			assertReinvokeCalls(t, mutator, 1)
		}
	})
}

func TestHandlePodMutation(t *testing.T) {
	t.Run("should call both mutators, initContainer and annotation added, no error", func(t *testing.T) {
		mutator1 := createSimplePodMutatorMock(t)
		mutator2 := createSimplePodMutatorMock(t)
		dk := getTestDynakube()
		podWebhook := createTestInjector([]dtwebhook.PodMutator{mutator1, mutator2}, false)
		mutationRequest := createTestMutationRequest(dk)
		podWebhook.recorder.Setup(mutationRequest)

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
		mutator1.AssertCalled(t, "Enabled", mutationRequest.BaseRequest)
		mutator1.AssertCalled(t, "Mutate", mock.Anything, mutationRequest)
		mutator2.AssertCalled(t, "Enabled", mutationRequest.BaseRequest)
		mutator2.AssertCalled(t, "Mutate", mock.Anything, mutationRequest)
	})
	t.Run("should call 1 webhook, 1 error, no initContainer and annotation", func(t *testing.T) {
		sadMutator := createFailPodMutatorMock(t)
		emptyMutator := webhookmock.NewPodMutator(t)
		dk := getTestDynakube()
		podWebhook := createTestInjector([]dtwebhook.PodMutator{sadMutator, emptyMutator}, false)
		mutationRequest := createTestMutationRequest(dk)
		podWebhook.recorder.Setup(mutationRequest)

		err := podWebhook.handlePodMutation(context.Background(), mutationRequest)
		require.Error(t, err)
		assert.NotNil(t, mutationRequest.InstallContainer)
		assert.Len(t, mutationRequest.Pod.Spec.InitContainers, 1)
		sadMutator.AssertCalled(t, "Enabled", mutationRequest.BaseRequest)
		sadMutator.AssertCalled(t, "Mutate", mock.Anything, mutationRequest)
	})
}

func TestHandlePodReinvocation(t *testing.T) {
	t.Run("should call both mutators, updated == true", func(t *testing.T) {
		mutator1 := createAlreadyInjectedPodMutatorMock(t)
		mutator2 := createAlreadyInjectedPodMutatorMock(t)
		dk := getTestDynakube()
		podWebhook := createTestInjector([]dtwebhook.PodMutator{mutator1, mutator2}, true)
		mutationRequest := createTestMutationRequestWithInjectedPod(dk)
		podWebhook.recorder.Setup(mutationRequest)

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
		noUpdateMutator := createNoUpdatePodMutatorMock(t)
		workingMutator := createAlreadyInjectedPodMutatorMock(t)
		dk := getTestDynakube()
		podWebhook := createTestInjector([]dtwebhook.PodMutator{noUpdateMutator, workingMutator}, true)
		mutationRequest := createTestMutationRequestWithInjectedPod(dk)
		podWebhook.recorder.Setup(mutationRequest)

		updated := podWebhook.handlePodReinvocation(mutationRequest)
		require.True(t, updated)
		noUpdateMutator.AssertCalled(t, "Enabled", mutationRequest.BaseRequest)
		noUpdateMutator.AssertCalled(t, "Reinvoke", mutationRequest.ToReinvocationRequest())
		workingMutator.AssertCalled(t, "Enabled", mutationRequest.BaseRequest)
		workingMutator.AssertCalled(t, "Reinvoke", mutationRequest.ToReinvocationRequest())
	})
	t.Run("should call webhook, no update", func(t *testing.T) {
		noUpdateMutator := createNoUpdatePodMutatorMock(t)
		dk := getTestDynakube()
		podWebhook := createTestInjector([]dtwebhook.PodMutator{noUpdateMutator}, true)
		mutationRequest := createTestMutationRequestWithInjectedPod(dk)
		podWebhook.recorder.Setup(mutationRequest)

		updated := podWebhook.handlePodReinvocation(mutationRequest)
		require.False(t, updated)
		noUpdateMutator.AssertCalled(t, "Enabled", mutationRequest.BaseRequest)
		noUpdateMutator.AssertCalled(t, "Reinvoke", mutationRequest.ToReinvocationRequest())
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

func assertMutateCalls(t *testing.T, mutator dtwebhook.PodMutator, expectedCalls int) {
	mock, ok := mutator.(*webhookmock.PodMutator)
	if !ok {
		t.Fatalf("assertPodMutatorCalls: webhook is not a mock")
	}

	mock.AssertNumberOfCalls(t, "Mutate", expectedCalls)
}

func assertReinvokeCalls(t *testing.T, mutator dtwebhook.PodMutator, expectedCalls int) {
	mock, ok := mutator.(*webhookmock.PodMutator)
	if !ok {
		t.Fatalf("assertPodMutatorCalls: webhook is not a mock")
	}

	mock.AssertNumberOfCalls(t, "Reinvoke", expectedCalls)
}

func createTestInjector(mutators []dtwebhook.PodMutator, isContainerInjected bool) *Injector {
	return &Injector{
		recorder:            events.NewRecorder(record.NewFakeRecorder(10)),
		webhookImage:        testImage,
		clusterID:           testClusterID,
		mutators:            mutators,
		isContainerInjected: func(c corev1.Container) bool { return isContainerInjected },
	}
}

func createSimplePodMutatorMock(t *testing.T) *webhookmock.PodMutator {
	t.Helper()

	mutator := webhookmock.NewPodMutator(t)
	mutator.On("Injected", mock.Anything).Return(false).Maybe() // It is a Maybe, because it is only checked at the very beginning
	mutator.On("Enabled", mock.Anything).Return(true)
	mutator.On("Mutate", mock.Anything, mock.Anything).Return(nil)

	return mutator
}

func createAlreadyInjectedPodMutatorMock(t *testing.T) *webhookmock.PodMutator {
	t.Helper()

	mutator := webhookmock.NewPodMutator(t)
	mutator.On("Injected", mock.Anything).Return(true).Maybe() // It is a Maybe, because if there are multiple mutators, the first "Injected" that returns true will break the loop -> Reinvoke
	mutator.On("Enabled", mock.Anything).Return(true)
	mutator.On("Reinvoke", mock.Anything).Return(true)

	return mutator
}

func createNoUpdatePodMutatorMock(t *testing.T) *webhookmock.PodMutator {
	t.Helper()

	mutator := webhookmock.NewPodMutator(t)
	mutator.On("Injected", mock.Anything).Return(true).Maybe() // It is a Maybe, because if there are multiple mutators, the first "Injected" that returns true will break the loop -> Reinvoke
	mutator.On("Enabled", mock.Anything).Return(true)
	mutator.On("Reinvoke", mock.Anything).Return(false)

	return mutator
}

func createFailPodMutatorMock(t *testing.T) *webhookmock.PodMutator {
	t.Helper()

	mutator := webhookmock.NewPodMutator(t)
	mutator.On("Injected", mock.Anything).Return(false).Maybe() // It is a Maybe, because it is only checked at the very beginning
	mutator.On("Enabled", mock.Anything).Return(true)
	mutator.On("Mutate", mock.Anything, mock.Anything).Return(errors.New("BOOM"))

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
			OneAgent: oneagent.Spec{
				ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{},
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

func getCloudNativeSpec(initResources *corev1.ResourceRequirements) oneagent.Spec {
	return oneagent.Spec{
		CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{
			AppInjectionSpec: oneagent.AppInjectionSpec{
				InitResources: initResources,
			},
		},
	}
}

func getTestPod() *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testPodName,
			Namespace: testNamespaceName,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:            "container",
					Image:           "alpine",
					SecurityContext: getTestSecurityContext(),
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

const testUser int64 = 420

func getTestSecurityContext() *corev1.SecurityContext {
	return &corev1.SecurityContext{
		RunAsUser:  ptr.To(testUser),
		RunAsGroup: ptr.To(testUser),
	}
}

func createTestMutationRequest(dk *dynakube.DynaKube) *dtwebhook.MutationRequest {
	return dtwebhook.NewMutationRequest(context.Background(), *getTestNamespace(), nil, getTestPod(), *dk)
}

func createTestMutationRequestWithInjectedPod(dk *dynakube.DynaKube) *dtwebhook.MutationRequest {
	return dtwebhook.NewMutationRequest(context.Background(), *getTestNamespace(), nil, getInjectedPod(), *dk)
}

func getInjectedPod() *corev1.Pod {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testPodName,
			Namespace: testNamespaceName,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:            "container",
					Image:           "alpine",
					SecurityContext: getTestSecurityContext(),
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
	installContainer := createInstallInitContainerBase("test", "test", pod, *getTestDynakube())
	pod.Spec.InitContainers = append(pod.Spec.InitContainers, *installContainer)

	return pod
}

func getTestNamespace() *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNamespaceName,
			Labels: map[string]string{
				dtwebhook.InjectionInstanceLabel: testDynakubeName,
			},
		},
	}
}

// TestDoubleInjection is special test case for making sure that we do not inject the init-container 2 times incase 1 of the mutators are skipped.
// The mutators are intentionally NOT mocked, as to mock them properly for this scenario you would need to basically reimplement them in the mock.
// This test is necessary as the current interface is not ready to handle the scenario properly.
// Scenario: OneAgent mutation is Enabled however needs to be skipped due to not meeting the requirements, so it needs to annotate but not fully inject
func TestDoubleInjection(t *testing.T) {
	noCommunicationHostDK := getTestDynakube()
	fakeClient := fake.NewClient(noCommunicationHostDK, getTestNamespace())
	podWebhook := &Injector{
		recorder:     events.NewRecorder(record.NewFakeRecorder(10)),
		webhookImage: testImage,
		clusterID:    testClusterID,
		mutators: []dtwebhook.PodMutator{
			oamutation.NewMutator(
				testImage,
				testClusterID,
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

	request := createTestMutationRequest(noCommunicationHostDK)
	request.Pod = pod

	// simulate initial mutation, annotations + init-container <== skip in case on communication hosts
	err := podWebhook.Handle(context.Background(), request)
	require.NoError(t, err)
	require.NotNil(t, findInstallContainer(pod.Spec.InitContainers))

	// adding communicationHost to the dynakube to make the scenario more complicated
	// it shouldn't try to mutate the pod because now it could be enabled, that is just asking for trouble.
	communicationHostDK := getTestDynakube()
	communicationHostDK.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts = []oneagent.CommunicationHostStatus{{Host: "test"}}
	fakeClient = fake.NewClient(communicationHostDK, getTestNamespace())
	podWebhook.mutators = []dtwebhook.PodMutator{
		oamutation.NewMutator(
			testImage,
			testClusterID,
			fakeClient,
			fakeClient,
		),
		metadata.NewMutator(
			testNamespaceName,
			fakeClient,
			fakeClient,
			fakeClient,
		),
	}

	// simulate a Reinvocation
	request = createTestMutationRequest(noCommunicationHostDK)
	request.Pod = pod
	err = podWebhook.Handle(context.Background(), request)
	require.NoError(t, err)
}

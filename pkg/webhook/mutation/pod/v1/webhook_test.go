package v1

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/startup"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/common/events"
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
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	expectedResult func(t *testing.T, err error, mutators []dtwebhook.PodMutator)
}

func TestMutator(t *testing.T) {
	tests := []mutatorTest{
		{
			name:     "not yet injected => mutate",
			mutators: []dtwebhook.PodMutator{createSimplePodMutatorMock(t), createSimplePodMutatorMock(t)},
			expectedResult: func(t *testing.T, err error, mutators []dtwebhook.PodMutator) {
				require.NoError(t, err)

				for _, mutator := range mutators {
					assertMutateCalls(t, mutator, 1)
				}
			},
		},
		{
			name:     "already injected => reinvoke",
			mutators: []dtwebhook.PodMutator{createAlreadyInjectedPodMutatorMock(t), createAlreadyInjectedPodMutatorMock(t)},
			expectedResult: func(t *testing.T, err error, mutators []dtwebhook.PodMutator) {
				require.NoError(t, err)

				for _, mutator := range mutators {
					assertReinvokeCalls(t, mutator, 1)
				}
			},
		},
		{
			name:     "fail => error",
			mutators: []dtwebhook.PodMutator{createFailPodMutatorMock(t)},
			expectedResult: func(t *testing.T, err error, mutators []dtwebhook.PodMutator) {
				require.Error(t, err)

				for _, mutator := range mutators {
					assertMutateCalls(t, mutator, 1)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			// merge test objects with the test pod
			podWebhook := createTestWebhook(test.mutators, nil)

			err := podWebhook.Handle(ctx, createTestMutationRequest(getTestDynakube()))
			test.expectedResult(t, err, test.mutators)
		})
	}
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

func createTestWebhook(mutators []dtwebhook.PodMutator, objects []client.Object) *Injector {
	return &Injector{
		recorder:     events.NewRecorder(record.NewFakeRecorder(10)),
		webhookImage: testImage,
		clusterID:    testClusterID,
		mutators:     mutators,
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
	mutator.On("Mutate", mock.Anything, mock.Anything).Return(errors.New("BOOM")).Maybe()
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

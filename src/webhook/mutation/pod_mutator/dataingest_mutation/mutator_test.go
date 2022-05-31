package dataingest_mutation

import (
	"context"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	dtingestendpoint "github.com/Dynatrace/dynatrace-operator/src/ingestendpoint"
	"github.com/Dynatrace/dynatrace-operator/src/mapper"
	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/src/webhook"
	"github.com/stretchr/testify/assert"
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

func TestEnabled(t *testing.T) {
	t.Run("turned off", func(t *testing.T) {
		mutator := createTestPodMutator(nil)
		pod := getTestPod(map[string]string{dtwebhook.AnnotationDataIngestInject: "false"})

		enabled := mutator.Enabled(pod)

		require.False(t, enabled)
	})
	t.Run("on by default", func(t *testing.T) {
		mutator := createTestPodMutator(nil)
		pod := getTestPod(nil)

		enabled := mutator.Enabled(pod)

		require.True(t, enabled)
	})
}

func TestInjected(t *testing.T) {
	t.Run("already marked", func(t *testing.T) {
		mutator := createTestPodMutator(nil)
		pod := getTestPod(map[string]string{dtwebhook.AnnotationDataIngestInjected: "true"})

		enabled := mutator.Injected(pod)

		require.True(t, enabled)
	})
	t.Run("fresh", func(t *testing.T) {
		mutator := createTestPodMutator(nil)
		pod := getTestPod(nil)

		enabled := mutator.Injected(pod)

		require.False(t, enabled)
	})
}

func TestMutate(t *testing.T) {
	t.Run("should mutate the pod and init container in the request", func(t *testing.T) {
		mutator := createTestPodMutator([]client.Object{getTestInitSecret()})
		request := createTestMutationRequest(getTestDynakube(), nil)
		initialNumberOfVolumesLen := len(request.Pod.Spec.Volumes)
		initialContainerVolumeMountsLen := len(request.Pod.Spec.Containers[0].VolumeMounts)
		initialAnnotationsLen := len(request.Pod.Annotations)

		err := mutator.Mutate(request)
		require.NoError(t, err)

		assert.Len(t, request.Pod.Spec.Volumes, initialNumberOfVolumesLen+2)
		assert.Len(t, request.Pod.Spec.Containers[0].VolumeMounts, initialContainerVolumeMountsLen+2)
		assert.Len(t, request.Pod.Annotations, initialAnnotationsLen+1)

		assert.Len(t, request.InstallContainer.Env, 3)
		assert.Len(t, request.InstallContainer.VolumeMounts, 1)
	})
}

func TestReinvoke(t *testing.T) {
	t.Run("should only mutate the containers in the pod", func(t *testing.T) {
		mutator := createTestPodMutator([]client.Object{getTestInitSecret()})
		request := createTestReinvocationRequest(getTestDynakube(), map[string]string{dtwebhook.AnnotationDataIngestInjected: "true"})

		initialContainerVolumeMountsLen := len(request.Pod.Spec.Containers[0].VolumeMounts)

		updated := mutator.Reinvoke(request)
		require.True(t, updated)

		assert.Len(t, request.Pod.Spec.Containers[0].VolumeMounts, initialContainerVolumeMountsLen+2)
	})
}

func TestEnsureDataIngestSecret(t *testing.T) {
	t.Run("shouldn't create init secret if already there", func(t *testing.T) {
		mutator := createTestPodMutator([]client.Object{getTestInitSecret()})
		request := createTestMutationRequest(getTestDynakube(), nil)

		err := mutator.ensureDataIngestSecret(request)
		require.NoError(t, err)
	})
}

func TestSetInjectedAnnotation(t *testing.T) {
	t.Run("should add annotation to nil map", func(t *testing.T) {
		pod := &corev1.Pod{}
		mutator := createTestPodMutator(nil)

		require.False(t, mutator.Injected(pod))
		setInjectedAnnotation(pod)
		require.Len(t, pod.Annotations, 1)
		require.True(t, mutator.Injected(pod))
	})
}

func TestContainerIsInjected(t *testing.T) {
	t.Run("is not injected", func(t *testing.T) {
		container := &corev1.Container{}

		isInjected := containerIsInjected(container)

		require.False(t, isInjected)
	})
	t.Run("is injected", func(t *testing.T) {
		container := &corev1.Container{
			VolumeMounts: []corev1.VolumeMount{
				{
					Name: workloadEnrichmentVolumeName,
				},
			},
		}

		isInjected := containerIsInjected(container)

		require.True(t, isInjected)
	})
}

func createTestMutationRequest(dynakube *dynatracev1beta1.DynaKube, annotations map[string]string) *dtwebhook.MutationRequest {
	return &dtwebhook.MutationRequest{
		Context:   context.TODO(),
		Pod:       getTestPod(annotations),
		Namespace: getTestNamespace(),
		DynaKube:  dynakube,
		InstallContainer: &corev1.Container{
			Name: dtwebhook.InstallContainerName,
		},
	}
}

func createTestReinvocationRequest(dynakube *dynatracev1beta1.DynaKube, annotations map[string]string) *dtwebhook.ReinvocationRequest {
	request := createTestMutationRequest(dynakube, annotations).ToReinvocationRequest()
	request.Pod.Spec.InitContainers = append(request.Pod.Spec.InitContainers, corev1.Container{Name: dtwebhook.InstallContainerName})
	return request
}

func createTestPodMutator(objects []client.Object) *DataIngestPodMutator {
	return &DataIngestPodMutator{
		client:           fake.NewClient(objects...),
		apiReader:        fake.NewClient(objects...),
		metaClient:       fake.NewClient(objects...),
		webhookNamespace: testNamespaceName,
	}
}

func getTestPod(annotations map[string]string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        testPodName,
			Namespace:   testNamespaceName,
			Annotations: annotations,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "container",
					Image: "alpine",
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "volume",
							MountPath: "/volume",
						},
					},
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

func getTestInitSecret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dtingestendpoint.SecretEndpointName,
			Namespace: testNamespaceName,
		},
	}
}

func getTestDynakube() *dynatracev1beta1.DynaKube {
	return &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testDynakubeName,
			Namespace: testNamespaceName,
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			OneAgent: dynatracev1beta1.OneAgentSpec{
				ApplicationMonitoring: &dynatracev1beta1.ApplicationMonitoringSpec{},
			},
			ActiveGate: dynatracev1beta1.ActiveGateSpec{
				Capabilities: []dynatracev1beta1.CapabilityDisplayName{dynatracev1beta1.MetricsIngestCapability.DisplayName},
			},
		},
	}
}

func getTestNamespace() *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNamespaceName,
			Labels: map[string]string{
				mapper.InstanceLabel: testDynakubeName,
			},
		},
	}
}

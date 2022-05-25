package oneagent_mutation

import (
	"context"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
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
	testImage         = "test-image"
	testClusterID     = "test-cluster-id"
	testPodName       = "test-pod"
	testNamespaceName = "test-namespace"
	testDynakubeName  = "test-dynakube"
)

func TestGetVolumeMode(t *testing.T) {
	t.Run("should return csi volume mode", func(t *testing.T) {
		mutator := createTestPodMutator(nil)

		assert.Equal(t, provisionedVolumeMode, mutator.getVolumeMode(getTestCSIDynakube()))
	})
	t.Run("should return empty volume mode", func(t *testing.T) {
		mutator := createTestPodMutator(nil)

		assert.Equal(t, installerVolumeMode, mutator.getVolumeMode(getTestDynakube()))
	})
}

func TestEnsureInitSecret(t *testing.T) {
	t.Run("shouldn't create init secret if already there", func(t *testing.T) {
		mutator := createTestPodMutator([]client.Object{getTestInitSecret()})
		request := createTestMutationRequest(getTestDynakube(), nil)

		err := mutator.ensureInitSecret(request)
		require.NoError(t, err)
	})
}

func TestMutate(t *testing.T) {
	t.Run("basic, should mutate the pod and init container in the request", func(t *testing.T) {
		mutator := createTestPodMutator([]client.Object{getTestInitSecret()})
		request := createTestMutationRequest(getTestDynakube(), nil)

		initialNumberOfContainerEnvsLen := len(request.Pod.Spec.Containers[0].Env)
		initialNumberOfVolumesLen := len(request.Pod.Spec.Volumes)
		initialContainerVolumeMountsLen := len(request.Pod.Spec.Containers[0].VolumeMounts)
		initialContainersLen := len(request.Pod.Spec.Containers)
		initialAnnotationsLen := len(request.Pod.Annotations)
		initialInitContainers := request.Pod.Spec.InitContainers

		err := mutator.Mutate(request)
		require.NoError(t, err)

		assert.Equal(t, initialNumberOfContainerEnvsLen+2, len(request.Pod.Spec.Containers[0].Env))
		assert.Equal(t, initialNumberOfVolumesLen+3, len(request.Pod.Spec.Volumes))
		assert.Equal(t, len(initialInitContainers), len(request.Pod.Spec.InitContainers)) // the init container should be added when in the PodMutator
		assert.Equal(t, initialContainerVolumeMountsLen+3, len(request.Pod.Spec.Containers[0].VolumeMounts))
		assert.Equal(t, initialAnnotationsLen+1, len(request.Pod.Annotations))
		assert.Equal(t, initialInitContainers, request.Pod.Spec.InitContainers)

		assert.Equal(t, 6+(initialContainersLen*2), len(request.InstallContainer.Env))
		assert.Equal(t, 2, len(request.InstallContainer.VolumeMounts))
	})
	t.Run("everything turned on, should mutate the pod and init container in the request", func(t *testing.T) {
		dynakube := getTestCSIDynakube()
		dynakube.Spec.Proxy = &dynatracev1beta1.DynaKubeProxy{Value: "test-proxy"}
		dynakube.Spec.NetworkZone = "test-network-zone"
		dynakube.Spec.ActiveGate = dynatracev1beta1.ActiveGateSpec{
			Capabilities:  []dynatracev1beta1.CapabilityDisplayName{dynatracev1beta1.KubeMonCapability.DisplayName},
			TlsSecretName: "super-secret",
		}
		dynakube.Annotations = map[string]string{
			dynatracev1beta1.AnnotationFeatureOneAgentInitialConnectRetry: "5",
		}
		mutator := createTestPodMutator([]client.Object{getTestInitSecret()})
		request := createTestMutationRequest(dynakube, nil)

		initialNumberOfContainerEnvsLen := len(request.Pod.Spec.Containers[0].Env)
		initialNumberOfVolumesLen := len(request.Pod.Spec.Volumes)
		initialContainerVolumeMountsLen := len(request.Pod.Spec.Containers[0].VolumeMounts)
		initialContainersLen := len(request.Pod.Spec.Containers)
		initialAnnotationsLen := len(request.Pod.Annotations)
		initialInitContainers := request.Pod.Spec.InitContainers

		err := mutator.Mutate(request)
		require.NoError(t, err)

		assert.Equal(t, initialNumberOfContainerEnvsLen+5, len(request.Pod.Spec.Containers[0].Env))
		assert.Equal(t, initialNumberOfVolumesLen+3, len(request.Pod.Spec.Volumes))
		assert.Equal(t, len(initialInitContainers), len(request.Pod.Spec.InitContainers)) // the init container should be added when in the PodMutator
		assert.Equal(t, initialContainerVolumeMountsLen+4, len(request.Pod.Spec.Containers[0].VolumeMounts))
		assert.Equal(t, initialAnnotationsLen+1, len(request.Pod.Annotations))
		assert.Equal(t, initialInitContainers, request.Pod.Spec.InitContainers)

		assert.Equal(t, 6+(initialContainersLen*2), len(request.InstallContainer.Env))
		assert.Equal(t, 2, len(request.InstallContainer.VolumeMounts))
	})
}

func TestReinvoke(t *testing.T) {
	t.Run("basic, should only mutate the containers in the pod", func(t *testing.T) {
		mutator := createTestPodMutator([]client.Object{getTestInitSecret()})
		request := createTestReinvocationRequest(getTestDynakube(), map[string]string{dtwebhook.AnnotationOneAgentInjected: "true"})

		initialNumberOfContainerEnvsLen := len(request.Pod.Spec.Containers[0].Env)
		initialNumberOfVolumesLen := len(request.Pod.Spec.Volumes)
		initialContainerVolumeMountsLen := len(request.Pod.Spec.Containers[0].VolumeMounts)
		initialContainersLen := len(request.Pod.Spec.Containers)
		initialAnnotationsLen := len(request.Pod.Annotations)

		updated := mutator.Reinvoke(request)
		require.True(t, updated)

		assert.Equal(t, initialNumberOfVolumesLen, len(request.Pod.Spec.Volumes))
		assert.Equal(t, initialAnnotationsLen, len(request.Pod.Annotations))

		assert.Equal(t, initialNumberOfContainerEnvsLen+2, len(request.Pod.Spec.Containers[0].Env))
		assert.Equal(t, initialContainerVolumeMountsLen+3, len(request.Pod.Spec.Containers[0].VolumeMounts))
		assert.Equal(t, initialContainersLen*2, len(request.Pod.Spec.InitContainers[1].Env)) // only add the new container specific env vars
	})
	t.Run("everything turned on, should only mutate the containers in the pod", func(t *testing.T) {
		dynakube := getTestCSIDynakube()
		dynakube.Spec.Proxy = &dynatracev1beta1.DynaKubeProxy{Value: "test-proxy"}
		dynakube.Spec.NetworkZone = "test-network-zone"
		dynakube.Spec.ActiveGate = dynatracev1beta1.ActiveGateSpec{
			Capabilities:  []dynatracev1beta1.CapabilityDisplayName{dynatracev1beta1.KubeMonCapability.DisplayName},
			TlsSecretName: "super-secret",
		}
		dynakube.Annotations = map[string]string{
			dynatracev1beta1.AnnotationFeatureOneAgentInitialConnectRetry: "5",
		}
		mutator := createTestPodMutator([]client.Object{getTestInitSecret()})
		request := createTestReinvocationRequest(dynakube, map[string]string{dtwebhook.AnnotationOneAgentInjected: "true"})

		initialNumberOfContainerEnvsLen := len(request.Pod.Spec.Containers[0].Env)
		initialNumberOfVolumesLen := len(request.Pod.Spec.Volumes)
		initialContainerVolumeMountsLen := len(request.Pod.Spec.Containers[0].VolumeMounts)
		initialContainersLen := len(request.Pod.Spec.Containers)
		initialAnnotationsLen := len(request.Pod.Annotations)

		updated := mutator.Reinvoke(request)
		require.True(t, updated)

		assert.Equal(t, initialNumberOfVolumesLen, len(request.Pod.Spec.Volumes))
		assert.Equal(t, initialAnnotationsLen, len(request.Pod.Annotations))

		assert.Equal(t, initialNumberOfContainerEnvsLen+5, len(request.Pod.Spec.Containers[0].Env))
		assert.Equal(t, initialContainerVolumeMountsLen+4, len(request.Pod.Spec.Containers[0].VolumeMounts))
		assert.Equal(t, initialContainersLen*2, len(request.Pod.Spec.InitContainers[1].Env)) // only add the new container specific env vars
	})
}

func createTestPodMutator(objects []client.Object) *OneAgentPodMutator {
	return &OneAgentPodMutator{
		client:           fake.NewClient(objects...),
		apiReader:        fake.NewClient(objects...),
		image:            testImage,
		clusterID:        testClusterID,
		webhookNamespace: testNamespaceName,
	}
}

func getTestInitSecret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dtwebhook.SecretConfigName,
			Namespace: testNamespaceName,
		},
	}
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

func getTestCSIDynakube() *dynatracev1beta1.DynaKube {
	return &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testDynakubeName,
			Namespace: testNamespaceName,
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			OneAgent: dynatracev1beta1.OneAgentSpec{
				CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{},
			},
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
		},
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

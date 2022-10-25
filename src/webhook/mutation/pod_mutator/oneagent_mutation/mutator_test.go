package oneagent_mutation

import (
	"context"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/config"
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

func TestEnabled(t *testing.T) {
	t.Run("turned off", func(t *testing.T) {
		mutator := createTestPodMutator(nil)
		request := createTestMutationRequest(nil, map[string]string{dtwebhook.AnnotationOneAgentInject: "false"}, getTestNamespace(nil))

		enabled := mutator.Enabled(request.BaseRequest)

		require.False(t, enabled)
	})
	t.Run("on by default", func(t *testing.T) {
		mutator := createTestPodMutator(nil)
		request := createTestMutationRequest(nil, nil, getTestNamespace(nil))

		enabled := mutator.Enabled(request.BaseRequest)

		require.True(t, enabled)
	})
	t.Run("off by feature flag", func(t *testing.T) {
		mutator := createTestPodMutator(nil)
		request := createTestMutationRequest(nil, nil, getTestNamespace(nil))
		request.DynaKube.Annotations = map[string]string{dynatracev1beta1.AnnotationFeatureAutomaticInjection: "false"}

		enabled := mutator.Enabled(request.BaseRequest)

		require.False(t, enabled)
	})
	t.Run("on with feature flag", func(t *testing.T) {
		mutator := createTestPodMutator(nil)
		request := createTestMutationRequest(nil, nil, getTestNamespace(nil))
		request.DynaKube.Annotations = map[string]string{dynatracev1beta1.AnnotationFeatureAutomaticInjection: "true"}

		enabled := mutator.Enabled(request.BaseRequest)

		require.True(t, enabled)
	})
}

func TestInjected(t *testing.T) {
	t.Run("already marked", func(t *testing.T) {
		mutator := createTestPodMutator(nil)
		request := createTestMutationRequest(nil, map[string]string{dtwebhook.AnnotationOneAgentInjected: "true"}, getTestNamespace(nil))

		enabled := mutator.Injected(request.BaseRequest)

		require.True(t, enabled)
	})
	t.Run("fresh", func(t *testing.T) {
		mutator := createTestPodMutator(nil)
		request := createTestMutationRequest(nil, nil, getTestNamespace(nil))

		enabled := mutator.Injected(request.BaseRequest)

		require.False(t, enabled)
	})
}

func TestGetVolumeMode(t *testing.T) {
	t.Run("should return csi volume mode", func(t *testing.T) {
		mutator := createTestPodMutator(nil)

		assert.Equal(t, string(config.AgentCsiMode), mutator.getVolumeMode(*getTestCSIDynakube()))
	})
	t.Run("should return empty volume mode", func(t *testing.T) {
		mutator := createTestPodMutator(nil)

		assert.Equal(t, string(config.AgentInstallerMode), mutator.getVolumeMode(*getTestDynakube()))
	})
}

func TestEnsureInitSecret(t *testing.T) {
	t.Run("shouldn't create init secret if already there", func(t *testing.T) {
		mutator := createTestPodMutator([]client.Object{getTestInitSecret()})
		request := createTestMutationRequest(getTestDynakube(), nil, getTestNamespace(nil))

		err := mutator.ensureInitSecret(request)
		require.NoError(t, err)
	})
}

func TestMutate(t *testing.T) {
	t.Run("basic, should mutate the pod and init container in the request", func(t *testing.T) {
		mutator := createTestPodMutator([]client.Object{getTestInitSecret()})
		request := createTestMutationRequest(getTestDynakube(), nil, getTestNamespace(nil))

		initialNumberOfContainerEnvsLen := len(request.Pod.Spec.Containers[0].Env)
		initialNumberOfVolumesLen := len(request.Pod.Spec.Volumes)
		initialContainerVolumeMountsLen := len(request.Pod.Spec.Containers[0].VolumeMounts)
		initialContainersLen := len(request.Pod.Spec.Containers)
		initialAnnotationsLen := len(request.Pod.Annotations)
		initialInitContainers := request.Pod.Spec.InitContainers

		err := mutator.Mutate(request)
		require.NoError(t, err)

		assert.Len(t, request.Pod.Spec.Containers[0].Env, initialNumberOfContainerEnvsLen+2)
		assert.Len(t, request.Pod.Spec.Volumes, initialNumberOfVolumesLen+3)
		assert.Equal(t, len(initialInitContainers), len(request.Pod.Spec.InitContainers)) // the init container should be added when in the PodMutator
		assert.Len(t, request.Pod.Spec.Containers[0].VolumeMounts, initialContainerVolumeMountsLen+3)
		assert.Len(t, request.Pod.Annotations, initialAnnotationsLen+1)
		assert.Equal(t, initialInitContainers, request.Pod.Spec.InitContainers)

		assert.Len(t, request.InstallContainer.Env, expectedBaseInitContainerEnvCount+(initialContainersLen*2))
		assert.Len(t, request.InstallContainer.VolumeMounts, 3)
	})
	t.Run("everything turned on, should mutate the pod and init container in the request", func(t *testing.T) {
		mutator := createTestPodMutator([]client.Object{getTestInitSecret()})
		request := createTestMutationRequest(getTestComplexDynakube(), nil, getTestNamespace(nil))

		initialNumberOfContainerEnvsLen := len(request.Pod.Spec.Containers[0].Env)
		initialNumberOfVolumesLen := len(request.Pod.Spec.Volumes)
		initialContainerVolumeMountsLen := len(request.Pod.Spec.Containers[0].VolumeMounts)
		initialContainersLen := len(request.Pod.Spec.Containers)
		initialAnnotationsLen := len(request.Pod.Annotations)
		initialInitContainers := request.Pod.Spec.InitContainers

		err := mutator.Mutate(request)
		require.NoError(t, err)

		assert.Len(t, request.Pod.Spec.Containers[0].Env, initialNumberOfContainerEnvsLen+6)
		assert.Len(t, request.Pod.Spec.Volumes, initialNumberOfVolumesLen+3)
		assert.Equal(t, len(initialInitContainers), len(request.Pod.Spec.InitContainers)) // the init container should be added when in the PodMutator
		assert.Len(t, request.Pod.Spec.Containers[0].VolumeMounts, initialContainerVolumeMountsLen+5)
		assert.Len(t, request.Pod.Annotations, initialAnnotationsLen+1)
		assert.Equal(t, initialInitContainers, request.Pod.Spec.InitContainers)

		assert.Len(t, request.InstallContainer.Env, expectedBaseInitContainerEnvCount+(initialContainersLen*2))
		assert.Len(t, request.InstallContainer.VolumeMounts, 3)
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

		assert.Len(t, request.Pod.Spec.Volumes, initialNumberOfVolumesLen)
		assert.Len(t, request.Pod.Annotations, initialAnnotationsLen)

		assert.Len(t, request.Pod.Spec.Containers[0].Env, initialNumberOfContainerEnvsLen+2)
		assert.Len(t, request.Pod.Spec.Containers[0].VolumeMounts, initialContainerVolumeMountsLen+3)
		assert.Len(t, request.Pod.Spec.InitContainers[1].Env, initialContainersLen*2)
	})
	t.Run("everything turned on, should only mutate the containers in the pod", func(t *testing.T) {
		mutator := createTestPodMutator([]client.Object{getTestInitSecret()})
		request := createTestReinvocationRequest(getTestComplexDynakube(), map[string]string{dtwebhook.AnnotationOneAgentInjected: "true"})

		initialNumberOfContainerEnvsLen := len(request.Pod.Spec.Containers[0].Env)
		initialNumberOfVolumesLen := len(request.Pod.Spec.Volumes)
		initialContainerVolumeMountsLen := len(request.Pod.Spec.Containers[0].VolumeMounts)
		initialContainersLen := len(request.Pod.Spec.Containers)
		initialAnnotationsLen := len(request.Pod.Annotations)

		updated := mutator.Reinvoke(request)
		require.True(t, updated)

		assert.Len(t, request.Pod.Spec.Volumes, initialNumberOfVolumesLen)
		assert.Len(t, request.Pod.Annotations, initialAnnotationsLen)

		assert.Len(t, request.Pod.Spec.Containers[0].Env, initialNumberOfContainerEnvsLen+6)
		assert.Len(t, request.Pod.Spec.Containers[0].VolumeMounts, initialContainerVolumeMountsLen+5)
		assert.Len(t, request.Pod.Spec.InitContainers[1].Env, initialContainersLen*2)
	})
	t.Run("no change ==> no update", func(t *testing.T) {
		mutator := createTestPodMutator([]client.Object{getTestInitSecret()})
		request := &dtwebhook.ReinvocationRequest{
			BaseRequest: &dtwebhook.BaseRequest{
				DynaKube: *getTestDynakube(),
				Pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{dtwebhook.AnnotationOneAgentInjected: "true"},
					},
				},
			},
		}
		updated := mutator.Reinvoke(request)
		require.False(t, updated)
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
			Name:      config.AgentInitSecretName,
			Namespace: testNamespaceName,
		},
	}
}

func createTestMutationRequest(dynakube *dynatracev1beta1.DynaKube, podAnnotations map[string]string, namespace corev1.Namespace) *dtwebhook.MutationRequest {
	if dynakube == nil {
		dynakube = &dynatracev1beta1.DynaKube{}
	}
	return dtwebhook.NewMutationRequest(
		context.TODO(),
		namespace,
		&corev1.Container{
			Name: dtwebhook.InstallContainerName,
		},
		getTestPod(podAnnotations),
		*dynakube,
	)
}

func createTestReinvocationRequest(dynakube *dynatracev1beta1.DynaKube, annotations map[string]string) *dtwebhook.ReinvocationRequest {
	request := createTestMutationRequest(dynakube, annotations, getTestNamespace(nil)).ToReinvocationRequest()
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

func getTestComplexDynakube() *dynatracev1beta1.DynaKube {
	dynakube := getTestCSIDynakube()
	dynakube.Spec.Proxy = &dynatracev1beta1.DynaKubeProxy{Value: "test-proxy"}
	dynakube.Spec.NetworkZone = "test-network-zone"
	dynakube.Spec.ActiveGate = dynatracev1beta1.ActiveGateSpec{
		Capabilities:  []dynatracev1beta1.CapabilityDisplayName{dynatracev1beta1.KubeMonCapability.DisplayName},
		TlsSecretName: "super-secret",
	}
	dynakube.Annotations = map[string]string{
		dynatracev1beta1.AnnotationFeatureOneAgentInitialConnectRetry: "5",
		dynatracev1beta1.AnnotationFeatureLabelVersionDetection:       "true",
	}
	return dynakube
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
					Name:  "main-container",
					Image: "alpine",
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "volume",
							MountPath: "/volume",
						},
					},
				},
				{
					Name:  "sidecar-container",
					Image: "nginx",
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
					Image: "curlimages/curl",
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

func getTestNamespace(annotations map[string]string) corev1.Namespace {
	return corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNamespaceName,
			Labels: map[string]string{
				dtwebhook.InjectionInstanceLabel: testDynakubeName,
			},
			Annotations: annotations,
		},
	}
}

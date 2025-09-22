package pod

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	handlermock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/webhook/mutation/pod/handler"
	webhookmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/webhook/mutation/pod/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const testUser int64 = 420

func getTestSecurityContext() *corev1.SecurityContext {
	return &corev1.SecurityContext{
		RunAsUser:  ptr.To(testUser),
		RunAsGroup: ptr.To(testUser),
	}
}

func TestCreateMutationRequestBase(t *testing.T) {
	t.Run("should create a mutation request", func(t *testing.T) {
		dk := getTestDynakube()
		podWebhook := createTestWebhook(t, webhookmock.NewMutator(t), webhookmock.NewMutator(t), handlermock.NewHandler(t), getTestNamespace(), getTestPod(), dk)
		mutationRequest, err := podWebhook.createMutationRequestBase(context.Background(), *createTestAdmissionRequest(getTestPod()))
		require.NoError(t, err)
		require.NotNil(t, mutationRequest)

		expected := createTestMutationRequest(dk)
		assert.Equal(t, expected.Pod.ObjectMeta, mutationRequest.Pod.ObjectMeta)
		assert.Equal(t, expected.Pod.Spec.Containers, mutationRequest.Pod.Spec.Containers)
		assert.Equal(t, expected.Pod.Spec.InitContainers, mutationRequest.Pod.Spec.InitContainers)
		assert.Equal(t, expected.DynaKube.ObjectMeta, mutationRequest.DynaKube.ObjectMeta)
		assert.Equal(t, expected.DynaKube.Spec, mutationRequest.DynaKube.Spec)
	})
}

func TestGetPodFromRequest(t *testing.T) {
	t.Run("should return the pod struct", func(t *testing.T) {
		podWebhook := createTestWebhook(t, webhookmock.NewMutator(t), webhookmock.NewMutator(t), handlermock.NewHandler(t))
		expected := getTestPod()

		pod, err := getPodFromRequest(*createTestAdmissionRequest(expected), podWebhook.decoder)
		require.NoError(t, err)
		assert.Equal(t, expected, pod)
	})
}

func TestGetNamespaceFromRequest(t *testing.T) {
	t.Run("should return the namespace struct", func(t *testing.T) {
		expected := getTestNamespace()
		podWebhook := createTestWebhook(t, webhookmock.NewMutator(t), webhookmock.NewMutator(t), handlermock.NewHandler(t), expected)

		namespace, err := getNamespaceFromRequest(context.Background(), podWebhook.apiReader, *createTestAdmissionRequest(getTestPod()))
		require.NoError(t, err)
		assert.Equal(t, expected.ObjectMeta, namespace.ObjectMeta)
	})
}

func TestGetDynakubeName(t *testing.T) {
	t.Run("should return the dynakube's name", func(t *testing.T) {
		namespace := getTestNamespace()
		dynakubeName, err := getDynakubeName(*namespace)
		require.NoError(t, err)
		assert.Equal(t, testDynakubeName, dynakubeName)
	})
}

func TestGetDynakube(t *testing.T) {
	t.Run("should return the dynakube struct", func(t *testing.T) {
		expected := getTestDynakube()
		podWebhook := createTestWebhook(t, webhookmock.NewMutator(t), webhookmock.NewMutator(t), handlermock.NewHandler(t), expected)

		dynakube, err := podWebhook.getDynakube(context.Background(), testDynakubeName)
		require.NoError(t, err)
		assert.Equal(t, expected.ObjectMeta, dynakube.ObjectMeta)
		assert.Equal(t, expected.Spec.OneAgent.CloudNativeFullStack, dynakube.Spec.OneAgent.CloudNativeFullStack)
	})
}

func createTestMutationRequest(dk *dynakube.DynaKube) *dtwebhook.MutationRequest {
	return dtwebhook.NewMutationRequest(context.Background(), *getTestNamespace(), nil, getTestPod(), *dk)
}

func createTestAdmissionRequest(pod *corev1.Pod) *admission.Request {
	basePodBytes, _ := json.Marshal(pod)

	return &admission.Request{
		AdmissionRequest: admissionv1.AdmissionRequest{
			Object: runtime.RawExtension{
				Raw: basePodBytes,
			},
			Namespace: testNamespaceName,
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

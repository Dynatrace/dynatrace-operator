package pod_mutator

import (
	"context"
	"encoding/json"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/address"
	"github.com/Dynatrace/dynatrace-operator/src/mapper"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/src/webhook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const testUser int64 = 420

var testSecurityContext = &corev1.SecurityContext{
	RunAsUser:  address.Of(testUser),
	RunAsGroup: address.Of(testUser),
}

func TestCreateMutationRequestBase(t *testing.T) {
	t.Run("should create a mutation request", func(t *testing.T) {
		dynakube := getTestDynakube()
		podWebhook := createTestWebhook(t,
			[]dtwebhook.PodMutator{},
			[]client.Object{
				getTestNamespace(),
				getTestPod(),
				dynakube,
			})
		mutationRequest, err := podWebhook.createMutationRequestBase(context.TODO(), *createTestAdmissionRequest(getTestPod()))
		require.NoError(t, err)
		require.NotNil(t, mutationRequest)

		expected := createTestMutationRequest(dynakube)
		assert.Equal(t, expected.Namespace.ObjectMeta, expected.Namespace.ObjectMeta)
		assert.Equal(t, expected.Pod.ObjectMeta, mutationRequest.Pod.ObjectMeta)
		assert.Equal(t, expected.Pod.Spec.Containers, mutationRequest.Pod.Spec.Containers)
		assert.Equal(t, expected.Pod.Spec.InitContainers, mutationRequest.Pod.Spec.InitContainers)
		assert.Equal(t, expected.DynaKube.ObjectMeta, mutationRequest.DynaKube.ObjectMeta)
		assert.Equal(t, expected.DynaKube.Spec, mutationRequest.DynaKube.Spec)
	})
}

func TestGetPodFromRequest(t *testing.T) {
	t.Run("should return the pod struct", func(t *testing.T) {
		podWebhook := createTestWebhook(t,
			[]dtwebhook.PodMutator{},
			[]client.Object{},
		)
		expected := getTestPod()

		pod, err := getPodFromRequest(*createTestAdmissionRequest(expected), *podWebhook.decoder)
		require.NoError(t, err)
		assert.Equal(t, expected, pod)
	})
}

func TestGetNamespaceFromRequest(t *testing.T) {
	t.Run("should return the namespace struct", func(t *testing.T) {
		expected := getTestNamespace()
		podWebhook := createTestWebhook(t,
			[]dtwebhook.PodMutator{},
			[]client.Object{expected},
		)

		namespace, err := getNamespaceFromRequest(context.TODO(), podWebhook.apiReader, *createTestAdmissionRequest(getTestPod()))
		require.NoError(t, err)
		assert.Equal(t, expected.ObjectMeta, namespace.ObjectMeta)
	})
}

func TestGetDynakubeName(t *testing.T) {
	t.Run("should return the dynakube's name", func(t *testing.T) {
		namespace := getTestNamespace()
		dynakubeName, err := getDynakubeName(namespace)
		require.NoError(t, err)
		assert.Equal(t, testDynakubeName, dynakubeName)
	})
}

func TestGetDynakube(t *testing.T) {
	t.Run("should return the dynakube struct", func(t *testing.T) {
		expected := getTestDynakube()
		podWebhook := createTestWebhook(t,
			[]dtwebhook.PodMutator{},
			[]client.Object{expected},
		)

		dynakube, err := podWebhook.getDynakube(context.TODO(), testDynakubeName)
		require.NoError(t, err)
		assert.Equal(t, expected.ObjectMeta, dynakube.ObjectMeta)
		assert.Equal(t, expected.Spec.OneAgent.CloudNativeFullStack, dynakube.Spec.OneAgent.CloudNativeFullStack)
	})
}

func createTestMutationRequest(dynakube *dynatracev1beta1.DynaKube) *dtwebhook.MutationRequest {
	return &dtwebhook.MutationRequest{
		Context:   context.TODO(),
		Pod:       getTestPod(),
		Namespace: getTestNamespace(),
		DynaKube:  dynakube,
	}
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
					SecurityContext: testSecurityContext,
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

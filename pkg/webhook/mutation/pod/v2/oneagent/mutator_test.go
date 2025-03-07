package oneagent

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/oneagent"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	oacommon "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/common/oneagent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testDynakubeName  = "test-dynakube"
	testNamespaceName = "test-namespace"
	testPodName       = "test-pod"
	customImage       = "custom-image"
)

func TestMutation(t *testing.T) {
	t.Run("should mutate the pod and init container in the request", func(t *testing.T) {
		mutator := createTestPodMutator()
		request := createTestMutationRequest(getTestDynakube(), nil, getTestNamespace(nil))

		err := mutator.Mutate(context.Background(), request)
		require.NoError(t, err)

		assert.Len(t, request.Pod.Spec.Containers[0].Env, 2)          // 1 deployment-metadata + 1 preload
		assert.Len(t, request.Pod.Spec.Volumes, 3)                    // from the pod + 2 from the mutator
		assert.Len(t, request.Pod.Spec.Containers[0].VolumeMounts, 3) // from the pod + 2 from the mutator
		assert.Len(t, request.InstallContainer.VolumeMounts, 2)       // bin, config
		assert.Equal(t, "true", request.Pod.Annotations[oacommon.AnnotationInjected])
	})
	t.Run("should not mutate the pod if the image is not set", func(t *testing.T) {
		mutator := createTestPodMutator()
		request := createTestMutationRequest(getTestDynakubeWithoutImage(), nil, getTestNamespace(nil))

		err := mutator.Mutate(context.Background(), request)
		require.NoError(t, err)

		assert.Empty(t, request.Pod.Spec.Containers[0].Env)
		assert.Len(t, request.Pod.Spec.Volumes, 1)
		assert.Len(t, request.Pod.Spec.Containers[0].VolumeMounts, 1)
		assert.Equal(t, "false", request.Pod.Annotations[oacommon.AnnotationInjected])
		assert.Equal(t, oacommon.UnknownCodeModuleReason, request.Pod.Annotations[oacommon.AnnotationReason])
	})
}

func createTestPodMutator(objects ...client.Object) *Mutator {
	return &Mutator{
		apiReader: fake.NewClient(objects...),
	}
}

func createTestMutationRequest(dk *dynakube.DynaKube, podAnnotations map[string]string, namespace corev1.Namespace) *dtwebhook.MutationRequest {
	if dk == nil {
		dk = &dynakube.DynaKube{}
	}

	return dtwebhook.NewMutationRequest(
		context.Background(),
		namespace,
		&corev1.Container{
			Name: dtwebhook.InstallContainerName,
		},
		getTestPod(podAnnotations),
		*dk,
	)
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

func getTestDynakube() *dynakube.DynaKube {
	return &dynakube.DynaKube{
		ObjectMeta: getTestDynakubeMeta(),
		Spec: dynakube.DynaKubeSpec{
			OneAgent: oneagent.Spec{
				ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{
					AppInjectionSpec: oneagent.AppInjectionSpec{
						CodeModulesImage: customImage,
					},
				},
			},
		},
		Status: getTestDynakubeStatus(),
	}
}

func getTestDynakubeWithoutImage() *dynakube.DynaKube {
	dk := getTestDynakube()
	dk.Spec.OneAgent.ApplicationMonitoring.AppInjectionSpec.CodeModulesImage = ""

	return dk
}

func getTestDynakubeMeta() metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      testDynakubeName,
		Namespace: testNamespaceName,
		Annotations: map[string]string{
			dynakube.AnnotationFeatureRemoteImageDownload: "true",
		},
	}
}

func getTestDynakubeStatus() dynakube.DynaKubeStatus {
	return dynakube.DynaKubeStatus{
		OneAgent: oneagent.Status{
			ConnectionInfoStatus: oneagent.ConnectionInfoStatus{
				ConnectionInfo: communication.ConnectionInfo{
					TenantUUID: "test-tenant-uuid",
				},
				CommunicationHosts: []oneagent.CommunicationHostStatus{
					{
						Protocol: "http",
						Host:     "dummyhost",
						Port:     666,
					},
				},
			},
		},
	}
}

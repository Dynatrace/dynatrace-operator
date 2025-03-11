package v2

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/common/events"
	oacommon "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/common/oneagent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
)

const (
	testNamespaceName = "test-namespace"
	testPodName       = "test-pod"
	testDynakubeName  = "test-dynakube"
	customImage       = "custom-image"
)

func TestHandle(t *testing.T) {
	ctx := context.Background()

	initSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.BootstrapperInitSecretName,
			Namespace: testNamespaceName,
		},
	}

	t.Run("no init secret => no injection + only annotation", func(t *testing.T) {
		injector := createTestInjector()
		injector.apiReader = fake.NewClient()

		request := createTestMutationRequest(getTestDynakube())

		err := injector.Handle(ctx, request)
		require.NoError(t, err)

		isInjected, ok := request.Pod.Annotations[oacommon.AnnotationInjected]
		require.True(t, ok)
		assert.Equal(t, "false", isInjected)

		reason, ok := request.Pod.Annotations[oacommon.AnnotationReason]
		require.True(t, ok)
		assert.Equal(t, NoBootstrapperConfigReason, reason)
	})

	t.Run("no codeModulesImage => no injection + only annotation", func(t *testing.T) {
		injector := createTestInjector()
		injector.apiReader = fake.NewClient(&initSecret)

		request := createTestMutationRequest(&dynakube.DynaKube{})

		err := injector.Handle(ctx, request)
		require.NoError(t, err)

		isInjected, ok := request.Pod.Annotations[oacommon.AnnotationInjected]
		require.True(t, ok)
		assert.Equal(t, "false", isInjected)

		reason, ok := request.Pod.Annotations[oacommon.AnnotationReason]
		require.True(t, ok)
		assert.Equal(t, NoCodeModulesImageReason, reason)
	})
}

func TestIsInjected(t *testing.T) {
	t.Run("init-container present == injected", func(t *testing.T) {
		injector := createTestInjector()

		assert.True(t, injector.isInjected(createTestMutationRequestWithInjectedPod(getTestDynakube())))
	})

	t.Run("init-container NOT present != injected", func(t *testing.T) {
		injector := createTestInjector()

		assert.False(t, injector.isInjected(createTestMutationRequest(getTestDynakube())))
	})
}

func createTestInjector() *Injector {
	return &Injector{
		recorder: events.NewRecorder(record.NewFakeRecorder(10)),
	}
}

func getTestDynakube() *dynakube.DynaKube {
	return &dynakube.DynaKube{
		ObjectMeta: getTestDynakubeMeta(),
		Spec: dynakube.DynaKubeSpec{
			OneAgent: getAppMonSpec(&testResourceRequirements),
		},
	}
}

var testResourceRequirements = corev1.ResourceRequirements{
	Limits: map[corev1.ResourceName]resource.Quantity{
		corev1.ResourceCPU:    resource.MustParse("100m"),
		corev1.ResourceMemory: resource.MustParse("100Mi"),
	},
}

func getTestDynakubeNoInitLimits() *dynakube.DynaKube {
	return &dynakube.DynaKube{
		ObjectMeta: getTestDynakubeMeta(),
		Spec: dynakube.DynaKubeSpec{
			OneAgent: getAppMonSpec(nil),
		},
	}
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

func getAppMonSpec(initResources *corev1.ResourceRequirements) oneagent.Spec {
	return oneagent.Spec{
		ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{
			AppInjectionSpec: oneagent.AppInjectionSpec{
				InitResources:    initResources,
				CodeModulesImage: customImage,
			}},
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
	installContainer := createInitContainerBase(pod, *getTestDynakube())
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

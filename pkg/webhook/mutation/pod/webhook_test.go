package pod

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/events"
	podwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	webhookmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/webhook/mutation/pod/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	testWebhookImage  = "test-wh-image"
	testNamespaceName = "test-namespace"
	testClusterID     = "test-cluster-id"
	testPodName       = "test-pod"
	testDynakubeName  = "test-dynakube"
)

func TestHandle(t *testing.T) {
	ctx := context.Background()

	t.Run("can't get NS ==> no inject, err in message", func(t *testing.T) {
		wh := createTestWebhook(t,
			webhookmock.NewMutator(t),
			webhookmock.NewMutator(t),
		)

		request := createTestAdmissionRequest(getTestPodWithInjectionDisabled())

		resp := wh.Handle(ctx, *request)
		require.NotNil(t, resp)
		assert.True(t, resp.Allowed)
		assert.NotEmpty(t, resp.Result.Message)
		assert.Contains(t, resp.Result.Message, "err")
	})

	t.Run("can't get DK ==> no inject, err in message", func(t *testing.T) {
		wh := createTestWebhook(t,
			webhookmock.NewMutator(t),
			webhookmock.NewMutator(t),
			getTestNamespace(),
		)

		request := createTestAdmissionRequest(getTestPodWithInjectionDisabled())

		resp := wh.Handle(ctx, *request)
		require.NotNil(t, resp)
		assert.True(t, resp.Allowed)
		assert.NotEmpty(t, resp.Result.Message)
		assert.Contains(t, resp.Result.Message, "err")
	})

	t.Run("DK name missing from NS but OLM ==> no inject, no err in message", func(t *testing.T) {
		ns := getTestNamespace()
		ns.Labels = map[string]string{}
		wh := createTestWebhook(t,
			webhookmock.NewMutator(t),
			webhookmock.NewMutator(t),
			ns,
		)
		wh.deployedViaOLM = true

		request := createTestAdmissionRequest(getTestPodWithInjectionDisabled())

		resp := wh.Handle(ctx, *request)
		require.NotNil(t, resp)
		assert.True(t, resp.Allowed)
		assert.NotEmpty(t, resp.Result.Message)
		assert.NotContains(t, resp.Result.Message, "err")
	})

	t.Run("DK name missing from NS ==> no inject, err in message", func(t *testing.T) {
		ns := getTestNamespace()
		ns.Labels = map[string]string{}
		wh := createTestWebhook(t,
			webhookmock.NewMutator(t),
			webhookmock.NewMutator(t),
			ns,
		)

		request := createTestAdmissionRequest(getTestPodWithInjectionDisabled())

		resp := wh.Handle(ctx, *request)
		require.NotNil(t, resp)
		assert.True(t, resp.Allowed)
		assert.NotEmpty(t, resp.Result.Message)
		assert.Contains(t, resp.Result.Message, "err")
	})

	t.Run("no inject annotation ==> no inject, empty patch", func(t *testing.T) {
		wh := createTestWebhook(t,
			webhookmock.NewMutator(t),
			webhookmock.NewMutator(t),
			getTestNamespace(),
			getTestDynakube(),
		)

		request := createTestAdmissionRequest(getTestPodWithInjectionDisabled())

		resp := wh.Handle(ctx, *request)
		require.NotNil(t, resp)
		assert.True(t, resp.Allowed)
		assert.Equal(t, admission.Patched(""), resp)
	})

	t.Run("no inject annotation (per container) ==> no inject, empty patch", func(t *testing.T) {
		wh := createTestWebhook(t,
			webhookmock.NewMutator(t),
			webhookmock.NewMutator(t),
			getTestNamespace(),
			getTestDynakube(),
		)

		request := createTestAdmissionRequest(getTestPodWithInjectionDisabledOnContainer())

		resp := wh.Handle(ctx, *request)
		require.NotNil(t, resp)
		assert.True(t, resp.Allowed)
		assert.Equal(t, admission.Patched(""), resp)
	})

	t.Run("OC debug pod ==> no inject", func(t *testing.T) {
		wh := createTestWebhook(t,
			webhookmock.NewMutator(t),
			webhookmock.NewMutator(t),
			getTestNamespace(),
			getTestDynakube(),
		)

		request := createTestAdmissionRequest(getTestPodWithOcDebugPodAnnotations())

		resp := wh.Handle(ctx, *request)
		require.NotNil(t, resp)
		assert.True(t, resp.Allowed)
		assert.Equal(t, admission.Patched(""), resp)
	})
}

func getTestPodWithInjectionDisabled() *corev1.Pod {
	pod := getTestPod()
	pod.Annotations = map[string]string{
		podwebhook.AnnotationDynatraceInject: "false",
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

func getTestPodWithInjectionDisabledOnContainer() *corev1.Pod {
	pod := getTestPod()
	pod.Annotations = map[string]string{}

	for _, c := range pod.Spec.Containers {
		pod.Annotations[podwebhook.AnnotationContainerInjection+"/"+c.Name] = "false"
	}

	return pod
}

func getTestWebhookPod(t *testing.T) corev1.Pod {
	t.Helper()

	return corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-webhook",
			Namespace: "test-namespace",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  dtwebhook.WebhookContainerName,
					Image: testWebhookImage,
				},
			},
		},
	}
}

func createTestWebhook(t *testing.T, oaMut, metaMut podwebhook.Mutator, objects ...client.Object) *webhook {
	t.Helper()

	decoder := admission.NewDecoder(scheme.Scheme)

	fakeClient := fake.NewClient(objects...)

	wh, err := newWebhook(fakeClient, fakeClient, fakeClient,
		events.NewRecorder(record.NewFakeRecorder(10)), decoder, getTestWebhookPod(t), false)

	require.NoError(t, err)

	wh.oaMutator = oaMut
	wh.metaMutator = metaMut

	return wh
}

func getTestDynakube() *dynakube.DynaKube {
	return &dynakube.DynaKube{
		ObjectMeta: getTestDynakubeMeta(),
		Spec: dynakube.DynaKubeSpec{
			OneAgent: getCloudNativeSpec(),
		},
	}
}

func getTestDynakubeMeta() metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      testDynakubeName,
		Namespace: testNamespaceName,
	}
}

func getCloudNativeSpec() oneagent.Spec {
	return oneagent.Spec{
		CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{
			AppInjectionSpec: oneagent.AppInjectionSpec{},
		},
	}
}

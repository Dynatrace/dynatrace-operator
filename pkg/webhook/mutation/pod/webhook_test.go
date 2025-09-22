package pod

import (
	"context"
	"errors"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/events"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/handler"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/handler/injection"
	podwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	handlermock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/webhook/mutation/pod/handler"
	webhookmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/webhook/mutation/pod/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gomodules.xyz/jsonpatch/v2"
	admissionv1 "k8s.io/api/admission/v1"
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
		wh := createTestWebhook(t, webhookmock.NewMutator(t), webhookmock.NewMutator(t), handlermock.NewHandler(t))

		request := createTestAdmissionRequest(getTestPodWithInjectionDisabled())

		resp := wh.Handle(ctx, *request)
		require.NotNil(t, resp)
		assert.True(t, resp.Allowed)
		assert.NotEmpty(t, resp.Result.Message)
		assert.Contains(t, resp.Result.Message, "err")
	})

	t.Run("can't get DK ==> no inject, err in message", func(t *testing.T) {
		wh := createTestWebhook(t, webhookmock.NewMutator(t), webhookmock.NewMutator(t), handlermock.NewHandler(t), getTestNamespace())

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
		wh := createTestWebhook(t, webhookmock.NewMutator(t), webhookmock.NewMutator(t), handlermock.NewHandler(t), ns)
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
		wh := createTestWebhook(t, webhookmock.NewMutator(t), webhookmock.NewMutator(t), handlermock.NewHandler(t), ns)

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
			handlermock.NewHandler(t),
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
			handlermock.NewHandler(t),
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
			handlermock.NewHandler(t),
			getTestNamespace(),
			getTestDynakube(),
		)

		request := createTestAdmissionRequest(getTestPodWithOcDebugPodAnnotations())

		resp := wh.Handle(ctx, *request)
		require.NotNil(t, resp)
		assert.True(t, resp.Allowed)
		assert.Equal(t, admission.Patched(""), resp)
	})
	t.Run("Arbitrary Error in OTLP handler ==> revert all modifications and include message", func(t *testing.T) {
		otlpHandler := handlermock.NewHandler(t)

		otlpHandler.On("Handle", mock.Anything).Return(errors.New("err"))
		wh := createTestWebhook(t,
			webhookmock.NewMutator(t),
			webhookmock.NewMutator(t),
			otlpHandler,
			getTestNamespace(),
			getTestDynakube(),
		)

		request := createTestAdmissionRequest(getTestPod())

		resp := wh.Handle(ctx, *request)
		require.NotNil(t, resp)
		assert.True(t, resp.Allowed)
		assert.Equal(t, admission.Patched("Failed to inject into pod: test-pod because err"), resp)
	})
	t.Run("MutatorError in OTLP handler ==> revert all modifications", func(t *testing.T) {
		otlpHandler := handlermock.NewHandler(t)

		annotated := false
		otlpHandler.On("Handle", mock.Anything).Return(&podwebhook.MutatorError{
			Err: errors.New("err"),
			Annotate: func(_ *corev1.Pod) {
				annotated = true
			},
		})
		wh := createTestWebhook(t,
			webhookmock.NewMutator(t),
			webhookmock.NewMutator(t),
			otlpHandler,
			getTestNamespace(),
			getTestDynakube(),
		)

		request := createTestAdmissionRequest(getTestPod())

		resp := wh.Handle(ctx, *request)
		require.NotNil(t, resp)

		assert.True(t, annotated)

		expected := admission.Response{
			// make sure no changes have been made to the pod due to the error returned by the handler
			Patches: make([]jsonpatch.JsonPatchOperation, 0),
			AdmissionResponse: admissionv1.AdmissionResponse{
				Allowed: true,
			},
		}
		assert.Equal(t, expected, resp)
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

func createTestWebhook(t *testing.T, oaMut, metaMut podwebhook.Mutator, otlpHandler handler.Handler, objects ...client.Object) *webhook {
	t.Helper()

	decoder := admission.NewDecoder(scheme.Scheme)

	fakeClient := fake.NewClient(objects...)

	wh, err := newWebhook(fakeClient, fakeClient, fakeClient,
		events.NewRecorder(record.NewFakeRecorder(10)), decoder, getTestWebhookPod(t), false)

	require.NoError(t, err)

	wh.injectionHandler = injection.New(
		fakeClient,
		fakeClient,
		wh.recorder,
		wh.webhookPodImage,
		false,
		metaMut,
		oaMut,
	)

	wh.otlpHandler = otlpHandler

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

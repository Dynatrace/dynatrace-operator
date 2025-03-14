package pod

import (
	"context"
	"errors"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/installconfig"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/common/events"
	webhookmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/webhook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	testImage         = "test-image"
	testNamespaceName = "test-namespace"
	testClusterID     = "test-cluster-id"
	testPodName       = "test-pod"
	testDynakubeName  = "test-dynakube"
)

func TestHandle(t *testing.T) {
	ctx := context.Background()

	t.Run("can't get NS ==> no inject, err in message", func(t *testing.T) {
		wh := createTestWebhook(
			webhookmock.NewPodInjector(t),
			webhookmock.NewPodInjector(t),
			[]client.Object{},
		)

		request := createTestAdmissionRequest(getTestPodWithInjectionDisabled())

		resp := wh.Handle(ctx, *request)
		require.NotNil(t, resp)
		assert.True(t, resp.Allowed)
		assert.NotEmpty(t, resp.Result.Message)
		assert.Contains(t, resp.Result.Message, "err")
	})

	t.Run("can't get DK ==> no inject, err in message", func(t *testing.T) {
		wh := createTestWebhook(
			webhookmock.NewPodInjector(t),
			webhookmock.NewPodInjector(t),
			[]client.Object{getTestNamespace()},
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
		wh := createTestWebhook(
			webhookmock.NewPodInjector(t),
			webhookmock.NewPodInjector(t),
			[]client.Object{ns},
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
		wh := createTestWebhook(
			webhookmock.NewPodInjector(t),
			webhookmock.NewPodInjector(t),
			[]client.Object{ns},
		)

		request := createTestAdmissionRequest(getTestPodWithInjectionDisabled())

		resp := wh.Handle(ctx, *request)
		require.NotNil(t, resp)
		assert.True(t, resp.Allowed)
		assert.NotEmpty(t, resp.Result.Message)
		assert.Contains(t, resp.Result.Message, "err")
	})

	t.Run("no inject annotation ==> no inject, empty patch", func(t *testing.T) {
		wh := createTestWebhook(
			webhookmock.NewPodInjector(t),
			webhookmock.NewPodInjector(t),
			[]client.Object{
				getTestNamespace(),
				getTestDynakube(),
			},
		)

		request := createTestAdmissionRequest(getTestPodWithInjectionDisabled())

		resp := wh.Handle(ctx, *request)
		require.NotNil(t, resp)
		assert.True(t, resp.Allowed)
		assert.Equal(t, admission.Patched(""), resp)
	})

	t.Run("no inject annotation (per container) ==> no inject, empty patch", func(t *testing.T) {
		wh := createTestWebhook(
			webhookmock.NewPodInjector(t),
			webhookmock.NewPodInjector(t),
			[]client.Object{
				getTestNamespace(),
				getTestDynakube(),
			},
		)

		request := createTestAdmissionRequest(getTestPodWithInjectionDisabledOnContainer())

		resp := wh.Handle(ctx, *request)
		require.NotNil(t, resp)
		assert.True(t, resp.Allowed)
		assert.Equal(t, admission.Patched(""), resp)
	})

	t.Run("OC debug pod ==> no inject", func(t *testing.T) {
		wh := createTestWebhook(
			webhookmock.NewPodInjector(t),
			webhookmock.NewPodInjector(t),
			[]client.Object{
				getTestNamespace(),
				getTestDynakube(),
			},
		)

		request := createTestAdmissionRequest(getTestPodWithOcDebugPodAnnotations())

		resp := wh.Handle(ctx, *request)
		require.NotNil(t, resp)
		assert.True(t, resp.Allowed)
		assert.Equal(t, admission.Patched(""), resp)
	})

	t.Run("no FF appmon-dk ==> v1 injector", func(t *testing.T) {
		v1Injector := webhookmock.NewPodInjector(t)
		v1Injector.On("Handle", mock.Anything, mock.Anything).Return(nil)
		wh := createTestWebhook(
			v1Injector,
			webhookmock.NewPodInjector(t),
			[]client.Object{
				getTestNamespace(),
				getTestDynakubeDefaultAppMon(),
			},
		)

		installconfig.SetModulesOverride(t, installconfig.Modules{CSIDriver: false})

		request := createTestAdmissionRequest(getTestPod())

		resp := wh.Handle(ctx, *request)
		require.NotNil(t, resp)
		assert.NotEqual(t, admission.Patched(""), resp)
	})

	t.Run("FF appmon-dk WITHOUT CSI ==> v2 injector", func(t *testing.T) {
		dk := getTestDynakubeDefaultAppMon()
		dk.Annotations = map[string]string{
			dynakube.AnnotationFeatureNodeImagePull: "true",
		}

		v2Injector := webhookmock.NewPodInjector(t)
		v2Injector.On("Handle", mock.Anything, mock.Anything).Return(nil)
		wh := createTestWebhook(
			webhookmock.NewPodInjector(t),
			v2Injector,
			[]client.Object{
				getTestNamespace(),
				dk,
			},
		)

		installconfig.SetModulesOverride(t, installconfig.Modules{CSIDriver: false})

		request := createTestAdmissionRequest(getTestPod())

		resp := wh.Handle(ctx, *request)
		require.NotNil(t, resp)
		assert.NotEqual(t, admission.Patched(""), resp)
	})

	t.Run("FF metadata-dk WITHOUT CSI ==> v1 injector", func(t *testing.T) {
		dk := getTestMetadataDynakube()
		dk.Annotations = map[string]string{
			dynakube.AnnotationFeatureNodeImagePull: "true",
		}

		v1Injector := webhookmock.NewPodInjector(t)
		v1Injector.On("Handle", mock.Anything, mock.Anything).Return(nil)
		wh := createTestWebhook(
			v1Injector,
			webhookmock.NewPodInjector(t),
			[]client.Object{
				getTestNamespace(),
				dk,
			},
		)

		installconfig.SetModulesOverride(t, installconfig.Modules{CSIDriver: false})

		request := createTestAdmissionRequest(getTestPod())

		resp := wh.Handle(ctx, *request)
		require.NotNil(t, resp)
		assert.NotEqual(t, admission.Patched(""), resp)
	})

	t.Run("FF appmon-dk WITH CSI ==> v1 injector", func(t *testing.T) {
		dk := getTestDynakubeDefaultAppMon()
		dk.Annotations = map[string]string{
			dynakube.AnnotationFeatureNodeImagePull: "true",
		}

		v1Injector := webhookmock.NewPodInjector(t)
		v1Injector.On("Handle", mock.Anything, mock.Anything).Return(nil)
		wh := createTestWebhook(
			v1Injector,
			webhookmock.NewPodInjector(t),
			[]client.Object{
				getTestNamespace(),
				dk,
			},
		)

		installconfig.SetModulesOverride(t, installconfig.Modules{CSIDriver: true})

		request := createTestAdmissionRequest(getTestPod())

		resp := wh.Handle(ctx, *request)
		require.NotNil(t, resp)
		assert.NotEqual(t, admission.Patched(""), resp)
	})

	t.Run("v1 injector error => silent error", func(t *testing.T) {
		v1Injector := webhookmock.NewPodInjector(t)
		v1Injector.On("Handle", mock.Anything, mock.Anything).Return(errors.New("BOOM"))
		wh := createTestWebhook(
			v1Injector,
			webhookmock.NewPodInjector(t),
			[]client.Object{
				getTestNamespace(),
				getTestDynakubeDefaultAppMon(),
			},
		)

		installconfig.SetModulesOverride(t, installconfig.Modules{CSIDriver: false})

		request := createTestAdmissionRequest(getTestPod())

		resp := wh.Handle(ctx, *request)
		require.NotNil(t, resp)
		assert.True(t, resp.Allowed)
		assert.NotEmpty(t, resp.Result.Message)
		assert.Contains(t, resp.Result.Message, "BOOM")
	})

	t.Run("v2 injector error => silent error", func(t *testing.T) {
		dk := getTestDynakubeDefaultAppMon()
		dk.Annotations = map[string]string{
			dynakube.AnnotationFeatureNodeImagePull: "true",
		}

		v2Injector := webhookmock.NewPodInjector(t)
		v2Injector.On("Handle", mock.Anything, mock.Anything).Return(errors.New("BOOM"))
		wh := createTestWebhook(
			webhookmock.NewPodInjector(t),
			v2Injector,
			[]client.Object{
				getTestNamespace(),
				dk,
			},
		)

		installconfig.SetModulesOverride(t, installconfig.Modules{CSIDriver: false})

		request := createTestAdmissionRequest(getTestPod())

		resp := wh.Handle(ctx, *request)
		require.NotNil(t, resp)
		assert.True(t, resp.Allowed)
		assert.NotEmpty(t, resp.Result.Message)
		assert.Contains(t, resp.Result.Message, "BOOM")
	})
}

func getTestPodWithInjectionDisabled() *corev1.Pod {
	pod := getTestPod()
	pod.Annotations = map[string]string{
		dtwebhook.AnnotationDynatraceInject: "false",
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
		pod.Annotations[dtwebhook.AnnotationContainerInjection+"/"+c.Name] = "false"
	}

	return pod
}

func createTestWebhook(v1, v2 dtwebhook.PodInjector, objects []client.Object) *webhook {
	decoder := admission.NewDecoder(scheme.Scheme)

	return &webhook{
		v1:               v1,
		v2:               v2,
		apiReader:        fake.NewClient(objects...),
		decoder:          decoder,
		webhookNamespace: testNamespaceName,
		recorder:         events.NewRecorder(record.NewFakeRecorder(10)),
	}
}

func getTestDynakube() *dynakube.DynaKube {
	return &dynakube.DynaKube{
		ObjectMeta: getTestDynakubeMeta(),
		Spec: dynakube.DynaKubeSpec{
			OneAgent: getCloudNativeSpec(),
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

func getTestMetadataDynakube() *dynakube.DynaKube {
	return &dynakube.DynaKube{
		ObjectMeta: getTestDynakubeMeta(),
		Spec: dynakube.DynaKubeSpec{
			MetadataEnrichment: dynakube.MetadataEnrichment{
				Enabled: ptr.To(true),
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

func getCloudNativeSpec() oneagent.Spec {
	return oneagent.Spec{
		CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{
			AppInjectionSpec: oneagent.AppInjectionSpec{},
		},
	}
}

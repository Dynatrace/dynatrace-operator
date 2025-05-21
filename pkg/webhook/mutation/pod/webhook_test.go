package pod

import (
	"context"
	"errors"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
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
			exp.OANodeImagePullKey: "true",
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
			exp.OANodeImagePullKey: "true",
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
			exp.OANodeImagePullKey: "true",
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
			exp.OANodeImagePullKey: "true",
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

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/bootstrapperconfig"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/installconfig"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/container"
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
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testNamespaceName = "test-namespace"
	testPodName       = "test-pod"
	testDynakubeName  = "test-dynakube"
	customImage       = "custom-image"
)

func TestIsEnabled(t *testing.T) {
	type testCase struct {
		title      string
		podMods    func(*corev1.Pod)
		dkMods     func(*dynakube.DynaKube)
		withCSI    bool
		withoutCSI bool
	}

	cases := []testCase{
		{
			title:      "nothing enabled => not enabled",
			podMods:    func(p *corev1.Pod) {},
			dkMods:     func(dk *dynakube.DynaKube) {},
			withCSI:    false,
			withoutCSI: false,
		},

		{
			title:   "only OA enabled, without FF => not enabled",
			podMods: func(p *corev1.Pod) {},
			dkMods: func(dk *dynakube.DynaKube) {
				dk.Spec.OneAgent.ApplicationMonitoring = &oneagent.ApplicationMonitoringSpec{}
			},
			withCSI:    false,
			withoutCSI: false,
		},

		{
			title:   "OA + FF enabled => enabled with no csi",
			podMods: func(p *corev1.Pod) {},
			dkMods: func(dk *dynakube.DynaKube) {
				dk.Spec.OneAgent.ApplicationMonitoring = &oneagent.ApplicationMonitoringSpec{}
				dk.Annotations = map[string]string{exp.OANodeImagePullKey: "true"}
			},
			withCSI:    false,
			withoutCSI: true,
		},
		{
			title: "OA + FF enabled + correct Volume-Type => enabled",
			podMods: func(p *corev1.Pod) {
				p.Annotations = map[string]string{oacommon.AnnotationVolumeType: oacommon.EphemeralVolumeType}
			},
			dkMods: func(dk *dynakube.DynaKube) {
				dk.Spec.OneAgent.ApplicationMonitoring = &oneagent.ApplicationMonitoringSpec{}
				dk.Annotations = map[string]string{exp.OANodeImagePullKey: "true"}
			},
			withCSI:    true,
			withoutCSI: true,
		},
		{
			title: "OA + FF enabled + incorrect Volume-Type => disabled",
			podMods: func(p *corev1.Pod) {
				p.Annotations = map[string]string{oacommon.AnnotationVolumeType: oacommon.CSIVolumeType}
			},
			dkMods: func(dk *dynakube.DynaKube) {
				dk.Spec.OneAgent.ApplicationMonitoring = &oneagent.ApplicationMonitoringSpec{}
				dk.Annotations = map[string]string{exp.OANodeImagePullKey: "true"}
			},
			withCSI:    false,
			withoutCSI: false,
		},
	}
	for _, test := range cases {
		t.Run(test.title, func(t *testing.T) {
			pod := &corev1.Pod{}
			test.podMods(pod)

			dk := &dynakube.DynaKube{}
			test.dkMods(dk)

			req := &dtwebhook.MutationRequest{BaseRequest: &dtwebhook.BaseRequest{Pod: pod, DynaKube: *dk}}

			assert.Equal(t, test.withCSI, IsEnabled(req))

			installconfig.SetModulesOverride(t, installconfig.Modules{CSIDriver: false})

			assert.Equal(t, test.withoutCSI, IsEnabled(req))
		})
	}
}

func TestHandle(t *testing.T) {
	ctx := context.Background()

	initSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.BootstrapperInitSecretName,
			Namespace: testNamespaceName,
		},
	}

	t.Run("no init secret + no init secret source => no injection + only annotation", func(t *testing.T) {
		injector := createTestInjectorBase()
		clt := fake.NewClient()
		injector.apiReader = clt

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

	t.Run("no init secret + source => replicate + inject", func(t *testing.T) {
		injector := createTestInjectorBase()
		request := createTestMutationRequest(getTestDynakube())

		source := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      bootstrapperconfig.GetSourceSecretName(request.DynaKube.Name),
				Namespace: request.DynaKube.Namespace,
			},
			Data: map[string][]byte{"data": []byte("beep")},
		}
		clt := fake.NewClient(&source)
		injector.kubeClient = clt
		injector.apiReader = clt

		err := injector.Handle(ctx, request)
		require.NoError(t, err)

		var replicated corev1.Secret
		err = clt.Get(context.Background(), client.ObjectKey{Name: consts.BootstrapperInitSecretName, Namespace: request.Namespace.Name}, &replicated)
		require.NoError(t, err)
		assert.Equal(t, source.Data, replicated.Data)

		isInjected, ok := request.Pod.Annotations[oacommon.AnnotationInjected]
		require.True(t, ok)
		assert.Equal(t, "true", isInjected)

		_, ok = request.Pod.Annotations[oacommon.AnnotationReason]
		require.False(t, ok)
	})

	t.Run("no codeModulesImage => no injection + only annotation", func(t *testing.T) {
		injector := createTestInjectorBase()
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

	t.Run("happy path", func(t *testing.T) {
		injector := createTestInjectorBase()
		injector.apiReader = fake.NewClient(&initSecret)

		request := createTestMutationRequest(getTestDynakube())

		err := injector.Handle(ctx, request)
		require.NoError(t, err)

		isInjected, ok := request.Pod.Annotations[oacommon.AnnotationInjected]
		require.True(t, ok)
		assert.Equal(t, "true", isInjected)

		_, ok = request.Pod.Annotations[oacommon.AnnotationReason]
		require.False(t, ok)

		installContainer := container.FindInitContainerInPodSpec(&request.Pod.Spec, dtwebhook.InstallContainerName)
		require.NotNil(t, installContainer)
		assert.Len(t, installContainer.Env, 3)
		assert.Len(t, installContainer.Args, 15)
	})
}

func TestIsInjected(t *testing.T) {
	t.Run("init-container present == injected", func(t *testing.T) {
		injector := createTestInjectorBase()

		assert.True(t, injector.isInjected(createTestMutationRequestWithInjectedPod(getTestDynakube())))
	})

	t.Run("init-container NOT present != injected", func(t *testing.T) {
		injector := createTestInjectorBase()

		assert.False(t, injector.isInjected(createTestMutationRequest(getTestDynakube())))
	})
}

func createTestInjectorBase() *Injector {
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
		Status: dynakube.DynaKubeStatus{
			KubernetesClusterMEID: "meid",
			KubeSystemUUID:        "systemuuid",
			KubernetesClusterName: "meidname",
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
			exp.OANodeImagePullKey: "true",
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
					Image:           "docker.io/php:fpm-stretch",
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

func TestIsCustomImageSet(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		request := dtwebhook.MutationRequest{
			BaseRequest: &dtwebhook.BaseRequest{
				DynaKube: *getTestDynakube(),
			},
		}

		assert.True(t, isCustomImageSet(&request))
	})
	t.Run("false, set annotations", func(t *testing.T) {
		request := dtwebhook.MutationRequest{
			BaseRequest: &dtwebhook.BaseRequest{
				DynaKube: *getTestDynakube(),
				Pod:      &corev1.Pod{},
			},
		}

		request.DynaKube.Spec.OneAgent.ApplicationMonitoring.CodeModulesImage = ""

		assert.False(t, isCustomImageSet(&request))
		assert.Equal(t, NoCodeModulesImageReason, request.Pod.Annotations[oacommon.AnnotationReason])
		assert.Equal(t, "false", request.Pod.Annotations[oacommon.AnnotationInjected])
	})
}

func TestSetDynatraceInjectedAnnotation(t *testing.T) {
	t.Run("add annotation", func(t *testing.T) {
		request := dtwebhook.MutationRequest{
			BaseRequest: &dtwebhook.BaseRequest{
				Pod: &corev1.Pod{},
			},
		}

		setDynatraceInjectedAnnotation(&request)

		require.Len(t, request.Pod.Annotations, 1)
		assert.Equal(t, "true", request.Pod.Annotations[dtwebhook.AnnotationDynatraceInjected])
	})

	t.Run("remove reason annotation", func(t *testing.T) {
		request := dtwebhook.MutationRequest{
			BaseRequest: &dtwebhook.BaseRequest{
				Pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							dtwebhook.AnnotationDynatraceReason: "beep",
						},
					},
				},
			},
		}

		setDynatraceInjectedAnnotation(&request)

		require.Len(t, request.Pod.Annotations, 1)
		assert.Equal(t, "true", request.Pod.Annotations[dtwebhook.AnnotationDynatraceInjected])
	})
}

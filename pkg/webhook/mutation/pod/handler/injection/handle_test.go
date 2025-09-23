package injection

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/bootstrapperconfig"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/container"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	webhookmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/webhook/mutation/pod/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestHandleImpl(t *testing.T) {
	initSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.BootstrapperInitSecretName,
			Namespace: testNamespaceName,
		},
	}

	certsSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.BootstrapperInitCertsSecretName,
			Namespace: testNamespaceName,
		},
	}

	t.Run("no init secret + no init secret source => no injection + only annotation", func(t *testing.T) {
		h := createTestHandler(webhookmock.NewMutator(t), webhookmock.NewMutator(t))

		request := createTestMutationRequest(getTestDynakube())

		err := h.Handle(request)
		require.NoError(t, err)

		isInjected, ok := request.Pod.Annotations[dtwebhook.AnnotationDynatraceInjected]
		require.True(t, ok)
		assert.Equal(t, "false", isInjected)

		reason, ok := request.Pod.Annotations[dtwebhook.AnnotationDynatraceReason]
		require.True(t, ok)
		assert.Equal(t, NoBootstrapperConfigReason, reason)
	})

	t.Run("no init secret and no certs + source (both) => replicate (both) + inject", func(t *testing.T) {
		request := createTestMutationRequest(getTestDynakubeWithAGCerts())

		source := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      bootstrapperconfig.GetSourceConfigSecretName(request.DynaKube.Name),
				Namespace: request.DynaKube.Namespace,
			},
			Data: map[string][]byte{"data": []byte("beep")},
		}
		sourceCerts := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      bootstrapperconfig.GetSourceCertsSecretName(request.DynaKube.Name),
				Namespace: request.DynaKube.Namespace,
			},
			Data: map[string][]byte{"certs": []byte("very secure")},
		}

		oaMutator := webhookmock.NewMutator(t)
		oaMutator.On("IsEnabled", mock.Anything).Return(true)
		oaMutator.On("Mutate", mock.Anything).Return(nil)

		metaMutator := webhookmock.NewMutator(t)
		metaMutator.On("IsEnabled", mock.Anything).Return(false)

		wh := createTestHandler(oaMutator, metaMutator, &source, &sourceCerts)

		err := wh.Handle(request)
		require.NoError(t, err)

		var replicated corev1.Secret
		err = wh.apiReader.Get(context.Background(), client.ObjectKey{Name: consts.BootstrapperInitSecretName, Namespace: request.Namespace.Name}, &replicated)
		require.NoError(t, err)
		assert.Equal(t, source.Data, replicated.Data)

		var replicatedCerts corev1.Secret
		err = wh.apiReader.Get(context.Background(), client.ObjectKey{Name: consts.BootstrapperInitCertsSecretName, Namespace: request.Namespace.Name}, &replicatedCerts)
		require.NoError(t, err)
		assert.Equal(t, sourceCerts.Data, replicatedCerts.Data)

		isInjected, ok := request.Pod.Annotations[dtwebhook.AnnotationDynatraceInjected]
		require.True(t, ok)
		assert.Equal(t, "true", isInjected)

		_, ok = request.Pod.Annotations[dtwebhook.AnnotationDynatraceReason]
		require.False(t, ok)
	})

	t.Run("no init and no certs, but don't replicate certs because we don't need it (AG is not enabled)", func(t *testing.T) {
		request := createTestMutationRequest(getTestDynakube())

		source := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      bootstrapperconfig.GetSourceConfigSecretName(request.DynaKube.Name),
				Namespace: request.DynaKube.Namespace,
			},
			Data: map[string][]byte{"data": []byte("beep")},
		}

		sourceCerts := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      bootstrapperconfig.GetSourceCertsSecretName(request.DynaKube.Name),
				Namespace: request.DynaKube.Namespace,
			},
			Data: map[string][]byte{"certs": []byte("very secure")},
		}

		oaMutator := webhookmock.NewMutator(t)
		oaMutator.On("IsEnabled", mock.Anything).Return(true)
		oaMutator.On("Mutate", mock.Anything).Return(nil)

		metaMutator := webhookmock.NewMutator(t)
		metaMutator.On("IsEnabled", mock.Anything).Return(true)
		metaMutator.On("Mutate", mock.Anything).Return(nil)

		wh := createTestHandler(oaMutator, metaMutator, &source, &sourceCerts)

		err := wh.Handle(request)
		require.NoError(t, err)

		var replicated corev1.Secret
		err = wh.apiReader.Get(context.Background(), client.ObjectKey{Name: consts.BootstrapperInitSecretName, Namespace: request.Namespace.Name}, &replicated)
		require.NoError(t, err)
		assert.Equal(t, source.Data, replicated.Data)

		var replicatedCerts corev1.Secret
		err = wh.apiReader.Get(context.Background(), client.ObjectKey{Name: consts.BootstrapperInitCertsSecretName, Namespace: request.Namespace.Name}, &replicatedCerts)
		require.Error(t, err)
		require.True(t, k8sErrors.IsNotFound(err))
		assert.Empty(t, replicatedCerts.Data)

		isInjected, ok := request.Pod.Annotations[dtwebhook.AnnotationDynatraceInjected]
		require.True(t, ok)
		assert.Equal(t, "true", isInjected)

		_, ok = request.Pod.Annotations[dtwebhook.AnnotationDynatraceReason]
		require.False(t, ok)
	})

	t.Run("happy path", func(t *testing.T) {
		oaMutator := webhookmock.NewMutator(t)
		oaMutator.On("IsEnabled", mock.Anything).Return(true)
		oaMutator.On("Mutate", mock.Anything).Return(nil)

		metaMutator := webhookmock.NewMutator(t)
		metaMutator.On("IsEnabled", mock.Anything).Return(true)
		metaMutator.On("Mutate", mock.Anything).Return(nil)

		h := createTestHandler(oaMutator, metaMutator, &initSecret, &certsSecret)

		request := createTestMutationRequest(getTestDynakube())

		err := h.Handle(request)
		require.NoError(t, err)

		isInjected, ok := request.Pod.Annotations[dtwebhook.AnnotationDynatraceInjected]
		require.True(t, ok)
		assert.Equal(t, "true", isInjected)

		_, ok = request.Pod.Annotations[dtwebhook.AnnotationDynatraceReason]
		require.False(t, ok)

		installContainer := container.FindInitContainerInPodSpec(&request.Pod.Spec, dtwebhook.InstallContainerName)
		require.NotNil(t, installContainer)
		assert.NotEmpty(t, installContainer.Env, 3)
		assert.NotEmpty(t, installContainer.Args, 15)
	})

	t.Run("happy path - nothing is enabled", func(t *testing.T) {
		oaMutator := webhookmock.NewMutator(t)
		oaMutator.On("IsEnabled", mock.Anything).Return(false)

		metaMutator := webhookmock.NewMutator(t)
		metaMutator.On("IsEnabled", mock.Anything).Return(false)

		h := createTestHandler(oaMutator, metaMutator, &initSecret, &certsSecret)

		request := createTestMutationRequest(getTestDynakube())

		err := h.Handle(request)
		require.NoError(t, err)

		isInjected, ok := request.Pod.Annotations[dtwebhook.AnnotationDynatraceInjected]
		require.True(t, ok)
		assert.Equal(t, "false", isInjected)

		reason, ok := request.Pod.Annotations[dtwebhook.AnnotationDynatraceReason]
		require.True(t, ok)
		assert.Equal(t, NoMutationNeededReason, reason)

		installContainer := container.FindInitContainerInPodSpec(&request.Pod.Spec, dtwebhook.InstallContainerName)
		require.Nil(t, installContainer)
	})

	t.Run("happy path - reinvoke", func(t *testing.T) {
		oaMutator := webhookmock.NewMutator(t)
		oaMutator.On("IsEnabled", mock.Anything).Return(true)
		oaMutator.On("Reinvoke", mock.Anything).Return(true)

		metaMutator := webhookmock.NewMutator(t)

		h := createTestHandler(oaMutator, metaMutator, &initSecret, &certsSecret)

		request := createTestMutationRequestWithInjectedPod(t, getTestDynakube())

		err := h.Handle(request)
		require.NoError(t, err)
	})
}

func TestIsInjected(t *testing.T) {
	t.Run("init-container present == injected", func(t *testing.T) {
		h := createTestHandler(nil, nil)

		assert.True(t, h.isInjected(createTestMutationRequestWithInjectedPod(t, getTestDynakube())))
	})

	t.Run("init-container NOT present != injected", func(t *testing.T) {
		h := createTestHandler(nil, nil)

		assert.False(t, h.isInjected(createTestMutationRequest(getTestDynakube())))
	})
}

func getTestDynakubeWithAGCerts() *dynakube.DynaKube {
	dk := getTestDynakube()
	dk.Spec.OneAgent.ApplicationMonitoring = &oneagent.ApplicationMonitoringSpec{}
	dk.Spec.ActiveGate = activegate.Spec{
		Capabilities: []activegate.CapabilityDisplayName{
			activegate.DynatraceAPICapability.DisplayName,
		},
		TLSSecretName: "ag-certs",
	}

	return dk
}

func createTestMutationRequestWithInjectedPod(t *testing.T, dk *dynakube.DynaKube) *dtwebhook.MutationRequest {
	t.Helper()

	return dtwebhook.NewMutationRequest(context.Background(), *getTestNamespace(), nil, getInjectedPod(t), *dk)
}

func getInjectedPod(t *testing.T) *corev1.Pod {
	t.Helper()

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

	h := createTestHandler(webhookmock.NewMutator(t), webhookmock.NewMutator(t))

	installContainer := h.createInitContainerBase(pod, *getTestDynakube())
	pod.Spec.InitContainers = append(pod.Spec.InitContainers, *installContainer)

	return pod
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

func createTestMutationRequest(dk *dynakube.DynaKube) *dtwebhook.MutationRequest {
	return dtwebhook.NewMutationRequest(context.Background(), *getTestNamespace(), nil, getTestPod(), *dk)
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

package metadata

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	dynatracev1beta2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testPodName          = "test-pod"
	testNamespaceName    = "test-namespace"
	testDynakubeName     = "test-dynakube"
	testApiUrl           = "http://test-endpoint/api"
	testWorkloadInfoName = "test-name"
	testWorkloadInfoKind = "test-kind"
)

func TestEnabled(t *testing.T) {
	t.Run("turned off", func(t *testing.T) {
		mutator := createTestPodMutator(nil)
		request := createTestMutationRequest(nil, map[string]string{dtwebhook.AnnotationMetadataEnrichmentInject: "false"})

		enabled := mutator.Enabled(request.BaseRequest)

		require.False(t, enabled)
	})
	t.Run("off by feature flag", func(t *testing.T) {
		mutator := createTestPodMutator(nil)
		request := createTestMutationRequest(nil, nil)
		request.DynaKube.Spec.MetadataEnrichment.Enabled = true
		request.DynaKube.Annotations = map[string]string{dynatracev1beta2.AnnotationFeatureAutomaticInjection: "false"}

		enabled := mutator.Enabled(request.BaseRequest)

		require.False(t, enabled)
	})
	t.Run("on with feature flag", func(t *testing.T) {
		mutator := createTestPodMutator(nil)
		dynakube := dynatracev1beta2.DynaKube{
			Spec: dynatracev1beta2.DynaKubeSpec{
				MetadataEnrichment: dynatracev1beta2.MetadataEnrichment{Enabled: true},
			},
		}
		request := createTestMutationRequest(&dynakube, nil)
		request.DynaKube.Annotations = map[string]string{dynatracev1beta2.AnnotationFeatureAutomaticInjection: "true"}

		enabled := mutator.Enabled(request.BaseRequest)

		require.True(t, enabled)
	})
}

func TestInjected(t *testing.T) {
	t.Run("already marked", func(t *testing.T) {
		mutator := createTestPodMutator(nil)
		request := createTestMutationRequest(nil, map[string]string{dtwebhook.AnnotationMetadataEnrichmentInjected: "true"})

		enabled := mutator.Injected(request.BaseRequest)

		require.True(t, enabled)
	})
	t.Run("fresh", func(t *testing.T) {
		mutator := createTestPodMutator(nil)
		request := createTestMutationRequest(nil, nil)

		enabled := mutator.Injected(request.BaseRequest)

		require.False(t, enabled)
	})
}

func TestMutate(t *testing.T) {
	t.Run("should mutate the pod and init container in the request", func(t *testing.T) {
		mutator := createTestPodMutator([]client.Object{getTestInitSecret()})
		request := createTestMutationRequest(getTestDynakube(), nil)
		initialNumberOfVolumesLen := len(request.Pod.Spec.Volumes)
		initialContainerVolumeMountsLen := len(request.Pod.Spec.Containers[0].VolumeMounts)
		initialAnnotationsLen := len(request.Pod.Annotations)

		err := mutator.Mutate(context.Background(), request)
		require.NoError(t, err)

		assert.Len(t, request.Pod.Spec.Volumes, initialNumberOfVolumesLen+2)
		assert.Len(t, request.Pod.Spec.Containers[0].VolumeMounts, initialContainerVolumeMountsLen+2)
		assert.Len(t, request.Pod.Annotations, initialAnnotationsLen+3)

		assert.Len(t, request.InstallContainer.Env, 3)
		assert.Len(t, request.InstallContainer.VolumeMounts, 1)
	})
}

func TestReinvoke(t *testing.T) {
	t.Run("should only mutate the containers in the pod", func(t *testing.T) {
		mutator := createTestPodMutator([]client.Object{getTestInitSecret()})
		request := createTestReinvocationRequest(getTestDynakube(), map[string]string{dtwebhook.AnnotationMetadataEnrichmentInjected: "true"})

		initialContainerVolumeMountsLen := len(request.Pod.Spec.Containers[0].VolumeMounts)

		updated := mutator.Reinvoke(request)
		require.True(t, updated)

		assert.Len(t, request.Pod.Spec.Containers[0].VolumeMounts, initialContainerVolumeMountsLen+2)
	})
	t.Run("no change ==> no update", func(t *testing.T) {
		mutator := createTestPodMutator([]client.Object{getTestInitSecret()})
		request := &dtwebhook.ReinvocationRequest{
			BaseRequest: &dtwebhook.BaseRequest{
				DynaKube: *getTestDynakube(),
				Pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{dtwebhook.AnnotationMetadataEnrichmentInjected: "true"},
					},
				},
			},
		}
		updated := mutator.Reinvoke(request)
		require.False(t, updated)
	})
}

func TestIngestEndpointSecret(t *testing.T) {
	t.Run("shouldn't create ingest secret if already there", func(t *testing.T) {
		mutator := createTestPodMutator([]client.Object{getTestInitSecret()})
		request := createTestMutationRequest(getTestDynakube(), nil)

		err := mutator.ensureIngestEndpointSecret(request)
		require.NoError(t, err)
	})

	t.Run("should create ingest secret", func(t *testing.T) {
		mutator := createTestPodMutator([]client.Object{getTestDynakube(), getTestTokensSecret()})
		request := createTestMutationRequest(getTestDynakube(), nil)

		err := mutator.ensureIngestEndpointSecret(request)
		require.NoError(t, err)

		var secret corev1.Secret
		err = mutator.apiReader.Get(context.Background(), client.ObjectKey{Name: consts.EnrichmentEndpointSecretName, Namespace: testNamespaceName}, &secret)
		require.NoError(t, err)
	})
}

func TestSetInjectedAnnotation(t *testing.T) {
	t.Run("should add annotation to nil map", func(t *testing.T) {
		request := createTestMutationRequest(nil, nil)
		mutator := createTestPodMutator(nil)

		require.False(t, mutator.Injected(request.BaseRequest))
		setInjectedAnnotation(request.Pod)
		require.Len(t, request.Pod.Annotations, 1)
		require.True(t, mutator.Injected(request.BaseRequest))
	})
}

func TestWorkloadAnnotations(t *testing.T) {
	t.Run("should add annotation to nil map", func(t *testing.T) {
		request := createTestMutationRequest(nil, nil)

		require.Equal(t, "not-found", maputils.GetField(request.Pod.Annotations, dtwebhook.AnnotationWorkloadName, "not-found"))
		setWorkloadAnnotations(request.Pod, &workloadInfo{name: testWorkloadInfoName, kind: testWorkloadInfoKind})
		require.Len(t, request.Pod.Annotations, 2)
		assert.Equal(t, testWorkloadInfoName, maputils.GetField(request.Pod.Annotations, dtwebhook.AnnotationWorkloadName, "not-found"))
		assert.Equal(t, testWorkloadInfoKind, maputils.GetField(request.Pod.Annotations, dtwebhook.AnnotationWorkloadKind, "not-found"))
	})
	t.Run("should lower case kind annotation", func(t *testing.T) {
		request := createTestMutationRequest(nil, nil)
		setWorkloadAnnotations(request.Pod, &workloadInfo{name: testWorkloadInfoName, kind: "SuperWorkload"})
		assert.Contains(t, request.Pod.Annotations, dtwebhook.AnnotationWorkloadKind)
		assert.Equal(t, "superworkload", request.Pod.Annotations[dtwebhook.AnnotationWorkloadKind])
	})
}

func TestContainerIsInjected(t *testing.T) {
	t.Run("is not injected", func(t *testing.T) {
		container := &corev1.Container{}

		isInjected := containerIsInjected(container)

		require.False(t, isInjected)
	})
	t.Run("is injected", func(t *testing.T) {
		container := &corev1.Container{
			VolumeMounts: []corev1.VolumeMount{
				{
					Name: workloadEnrichmentVolumeName,
				},
			},
		}

		isInjected := containerIsInjected(container)

		require.True(t, isInjected)
	})
}

func createTestMutationRequest(dynakube *dynatracev1beta2.DynaKube, annotations map[string]string) *dtwebhook.MutationRequest {
	if dynakube == nil {
		dynakube = &dynatracev1beta2.DynaKube{}
	}

	return dtwebhook.NewMutationRequest(
		context.Background(),
		*getTestNamespace(),
		&corev1.Container{
			Name: dtwebhook.InstallContainerName,
		},
		getTestPod(annotations),
		*dynakube,
	)
}

func createTestReinvocationRequest(dynakube *dynatracev1beta2.DynaKube, annotations map[string]string) *dtwebhook.ReinvocationRequest {
	request := createTestMutationRequest(dynakube, annotations).ToReinvocationRequest()
	request.Pod.Spec.InitContainers = append(request.Pod.Spec.InitContainers, corev1.Container{Name: dtwebhook.InstallContainerName})

	return request
}

func createTestPodMutator(objects []client.Object) *Mutator {
	fakeClient := fake.NewClient(objects...)

	return &Mutator{
		client:           fakeClient,
		apiReader:        fakeClient,
		metaClient:       fakeClient,
		webhookNamespace: testNamespaceName,
	}
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
					Name:  "container",
					Image: "alpine",
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

func getTestInitSecret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.EnrichmentEndpointSecretName,
			Namespace: testNamespaceName,
		},
	}
}

func getTestTokensSecret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testDynakubeName,
			Namespace: testNamespaceName,
		},
		Data: map[string][]byte{
			dtclient.DataIngestToken: []byte("test"),
		},
	}
}

func getTestDynakube() *dynatracev1beta2.DynaKube {
	return &dynatracev1beta2.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testDynakubeName,
			Namespace: testNamespaceName,
		},
		Spec: dynatracev1beta2.DynaKubeSpec{
			APIURL: testApiUrl,
			OneAgent: dynatracev1beta2.OneAgentSpec{
				ApplicationMonitoring: &dynatracev1beta2.ApplicationMonitoringSpec{},
			},
			ActiveGate: dynatracev1beta2.ActiveGateSpec{
				Capabilities: []dynatracev1beta2.CapabilityDisplayName{dynatracev1beta2.MetricsIngestCapability.DisplayName},
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

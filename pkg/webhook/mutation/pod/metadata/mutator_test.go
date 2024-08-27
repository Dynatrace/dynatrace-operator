package metadata

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
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
	testPodName             = "test-pod"
	testNamespaceName       = "test-namespace"
	testDynakubeName        = "test-dynakube"
	testApiUrl              = "http://test-endpoint/api"
	testWorkloadInfoName    = "test-name"
	testWorkloadInfoKind    = "test-kind"
	testLabelKeyMatching    = "inject"
	testLabelKeyNotMatching = "do-not-inject"
	testLabelValue          = "into-this-ns"
)

func TestEnabled(t *testing.T) {
	t.Run("turned off", func(t *testing.T) {
		mutator := createTestPodMutator(nil)
		request := createTestMutationRequest(nil, map[string]string{dtwebhook.AnnotationMetadataEnrichmentInject: "false"}, false)

		enabled := mutator.Enabled(request.BaseRequest)

		require.False(t, enabled)
	})
	t.Run("off by feature flag", func(t *testing.T) {
		mutator := createTestPodMutator(nil)
		request := createTestMutationRequest(nil, nil, false)
		request.DynaKube.Spec.MetadataEnrichment.Enabled = true
		request.DynaKube.Annotations = map[string]string{dynakube.AnnotationFeatureAutomaticInjection: "false"}

		enabled := mutator.Enabled(request.BaseRequest)

		require.False(t, enabled)
	})
	t.Run("on with feature flag", func(t *testing.T) {
		mutator := createTestPodMutator(nil)
		dk := dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				MetadataEnrichment: dynakube.MetadataEnrichment{Enabled: true},
			},
		}
		request := createTestMutationRequest(&dk, nil, false)
		request.DynaKube.Annotations = map[string]string{dynakube.AnnotationFeatureAutomaticInjection: "true"}

		enabled := mutator.Enabled(request.BaseRequest)

		require.True(t, enabled)
	})
	t.Run("on with namespaceselector", func(t *testing.T) {
		mutator := createTestPodMutator(nil)
		dk := dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				MetadataEnrichment: dynakube.MetadataEnrichment{
					Enabled: true,
					NamespaceSelector: metav1.LabelSelector{
						MatchLabels: map[string]string{
							testLabelKeyMatching: testLabelValue,
						},
					},
				},
			},
		}
		request := createTestMutationRequest(&dk, nil, true)
		request.DynaKube.Annotations = map[string]string{dynakube.AnnotationFeatureAutomaticInjection: "true"}

		enabled := mutator.Enabled(request.BaseRequest)

		require.True(t, enabled)
	})
	t.Run("off due to mismatching namespaceselector", func(t *testing.T) {
		mutator := createTestPodMutator(nil)
		dk := dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				MetadataEnrichment: dynakube.MetadataEnrichment{
					Enabled: true,
					NamespaceSelector: metav1.LabelSelector{
						MatchLabels: map[string]string{
							testLabelKeyNotMatching: testLabelValue,
						},
					},
				},
			},
		}
		request := createTestMutationRequest(&dk, nil, true)
		request.DynaKube.Annotations = map[string]string{dynakube.AnnotationFeatureAutomaticInjection: "true"}

		enabled := mutator.Enabled(request.BaseRequest)

		require.False(t, enabled)
	})
}

func TestInjected(t *testing.T) {
	t.Run("already marked", func(t *testing.T) {
		mutator := createTestPodMutator(nil)
		request := createTestMutationRequest(nil, map[string]string{dtwebhook.AnnotationMetadataEnrichmentInjected: "true"}, false)

		enabled := mutator.Injected(request.BaseRequest)

		require.True(t, enabled)
	})
	t.Run("fresh", func(t *testing.T) {
		mutator := createTestPodMutator(nil)
		request := createTestMutationRequest(nil, nil, false)

		enabled := mutator.Injected(request.BaseRequest)

		require.False(t, enabled)
	})
}

func TestMutate(t *testing.T) {
	t.Run("should mutate the pod and init container in the request", func(t *testing.T) {
		mutator := createTestPodMutator([]client.Object{getTestInitSecret()})
		request := createTestMutationRequest(getTestDynakube(), nil, false)
		initialNumberOfVolumesLen := len(request.Pod.Spec.Volumes)
		initialContainerVolumeMountsLen := len(request.Pod.Spec.Containers[0].VolumeMounts)
		initialAnnotationsLen := len(request.Pod.Annotations)

		err := mutator.Mutate(context.Background(), request)
		require.NoError(t, err)

		assert.Len(t, request.Pod.Spec.Volumes, initialNumberOfVolumesLen+2)
		assert.Len(t, request.Pod.Spec.Containers[0].VolumeMounts, initialContainerVolumeMountsLen+3)
		assert.Len(t, request.Pod.Annotations, initialAnnotationsLen+3)

		assert.Len(t, request.InstallContainer.Env, 4)
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

		assert.Len(t, request.Pod.Spec.Containers[0].VolumeMounts, initialContainerVolumeMountsLen+3)
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
		request := createTestMutationRequest(getTestDynakube(), nil, false)

		err := mutator.ensureIngestEndpointSecret(request)
		require.NoError(t, err)
	})

	t.Run("should create ingest secret", func(t *testing.T) {
		mutator := createTestPodMutator([]client.Object{getTestDynakube(), getTestTokensSecret()})
		request := createTestMutationRequest(getTestDynakube(), nil, false)

		err := mutator.ensureIngestEndpointSecret(request)
		require.NoError(t, err)

		var secret corev1.Secret
		err = mutator.apiReader.Get(context.Background(), client.ObjectKey{Name: consts.EnrichmentEndpointSecretName, Namespace: testNamespaceName}, &secret)
		require.NoError(t, err)
	})
}

func TestSetInjectedAnnotation(t *testing.T) {
	t.Run("should add annotation to nil map", func(t *testing.T) {
		request := createTestMutationRequest(nil, nil, false)
		mutator := createTestPodMutator(nil)

		require.False(t, mutator.Injected(request.BaseRequest))
		setInjectedAnnotation(request.Pod)
		require.Len(t, request.Pod.Annotations, 1)
		require.True(t, mutator.Injected(request.BaseRequest))
	})
}

func TestCopyMetadataFromNamespace(t *testing.T) {
	t.Run("should copy annotations not labels with prefix from namespace to pod", func(t *testing.T) {
		mutator := createTestPodMutator(nil)
		request := createTestMutationRequest(nil, nil, false)
		request.Namespace.Labels = map[string]string{
			dynakube.MetadataPrefix + "nocopyoflabels": "nocopyoflabels",
			"test-label": "test-value",
		}
		request.Namespace.Annotations = map[string]string{
			dynakube.MetadataPrefix + "copyofannotations": "copyofannotations",
			"test-annotation": "test-value",
		}

		require.False(t, mutator.Injected(request.BaseRequest))
		copyMetadataFromNamespace(request.Pod, request.Namespace, request.DynaKube)
		require.Len(t, request.Pod.Annotations, 1)
		require.Empty(t, request.Pod.Labels)
		require.Equal(t, "copyofannotations", request.Pod.Annotations[dynakube.MetadataPrefix+"copyofannotations"])
	})

	t.Run("should copy all labels and annotations defined without override", func(t *testing.T) {
		mutator := createTestPodMutator(nil)
		request := createTestMutationRequest(nil, nil, false)
		request.Namespace.Labels = map[string]string{
			dynakube.MetadataPrefix + "nocopyoflabels":   "nocopyoflabels",
			dynakube.MetadataPrefix + "copyifruleexists": "copyifruleexists",
			"test-label": "test-value",
		}
		request.Namespace.Annotations = map[string]string{
			dynakube.MetadataPrefix + "copyofannotations": "copyofannotations",
			"test-annotation": "test-value",
		}

		request.DynaKube.Status.MetadataEnrichment.Rules = []dynakube.EnrichmentRule{
			{
				Type:   dynakube.EnrichmentAnnotationRule,
				Source: "test-annotation",
				Target: "dt.test-annotation",
			},
			{
				Type:   dynakube.EnrichmentLabelRule,
				Source: "test-label",
				Target: "test-label",
			},
			{
				Type:   dynakube.EnrichmentLabelRule,
				Source: dynakube.MetadataPrefix + "copyifruleexists",
				Target: "dt.copyifruleexists",
			},
			{
				Type:   dynakube.EnrichmentLabelRule,
				Source: "does-not-exist-in-namespace",
				Target: "dt.does-not-exist-in-namespace",
			},
		}
		request.Pod.Annotations = map[string]string{
			dynakube.MetadataPrefix + "copyofannotations": "do-not-overwrite",
		}

		require.False(t, mutator.Injected(request.BaseRequest))
		copyMetadataFromNamespace(request.Pod, request.Namespace, request.DynaKube)
		require.Len(t, request.Pod.Annotations, 4)
		require.Empty(t, request.Pod.Labels)

		require.Equal(t, "do-not-overwrite", request.Pod.Annotations[dynakube.MetadataPrefix+"copyofannotations"])
		require.Equal(t, "copyifruleexists", request.Pod.Annotations[dynakube.MetadataPrefix+"dt.copyifruleexists"])

		require.Equal(t, "test-value", request.Pod.Annotations[dynakube.MetadataPrefix+"dt.test-annotation"])
		require.Equal(t, "test-value", request.Pod.Annotations[dynakube.MetadataPrefix+"test-label"])
	})

	t.Run("are custom rule types handled correctly", func(t *testing.T) {
		mutator := createTestPodMutator(nil)
		request := createTestMutationRequest(nil, nil, false)
		request.Namespace.Labels = map[string]string{
			"test":  "test-label-value",
			"test2": "test-label-value2",
			"test3": "test-label-value3",
		}
		request.Namespace.Annotations = map[string]string{
			"test":  "test-annotation-value",
			"test2": "test-annotation-value2",
			"test3": "test-annotation-value3",
		}

		request.DynaKube.Status.MetadataEnrichment.Rules = []dynakube.EnrichmentRule{
			{
				Type:   dynakube.EnrichmentLabelRule,
				Source: "test",
				Target: "dt.test-label",
			},
			{
				Type:   dynakube.EnrichmentAnnotationRule,
				Source: "test2",
				Target: "dt.test-annotation",
			},
			{
				Type:   dynakube.EnrichmentAnnotationRule,
				Source: "test3",
				Target: "", // mapping missing => rule ignored
			},
		}

		require.False(t, mutator.Injected(request.BaseRequest))
		copyMetadataFromNamespace(request.Pod, request.Namespace, request.DynaKube)
		require.Len(t, request.Pod.Annotations, 2)
		require.Empty(t, request.Pod.Labels)
		require.Equal(t, "test-label-value", request.Pod.Annotations[dynakube.MetadataPrefix+"dt.test-label"])
		require.Equal(t, "test-annotation-value2", request.Pod.Annotations[dynakube.MetadataPrefix+"dt.test-annotation"])
	})
}

func TestWorkloadAnnotations(t *testing.T) {
	t.Run("should add annotation to nil map", func(t *testing.T) {
		request := createTestMutationRequest(nil, nil, false)

		require.Equal(t, "not-found", maputils.GetField(request.Pod.Annotations, dtwebhook.AnnotationWorkloadName, "not-found"))
		setWorkloadAnnotations(request.Pod, &workloadInfo{name: testWorkloadInfoName, kind: testWorkloadInfoKind})
		require.Len(t, request.Pod.Annotations, 2)
		assert.Equal(t, testWorkloadInfoName, maputils.GetField(request.Pod.Annotations, dtwebhook.AnnotationWorkloadName, "not-found"))
		assert.Equal(t, testWorkloadInfoKind, maputils.GetField(request.Pod.Annotations, dtwebhook.AnnotationWorkloadKind, "not-found"))
	})
	t.Run("should lower case kind annotation", func(t *testing.T) {
		request := createTestMutationRequest(nil, nil, false)
		setWorkloadAnnotations(request.Pod, &workloadInfo{name: testWorkloadInfoName, kind: "SuperWorkload"})
		assert.Contains(t, request.Pod.Annotations, dtwebhook.AnnotationWorkloadKind)
		assert.Equal(t, "superworkload", request.Pod.Annotations[dtwebhook.AnnotationWorkloadKind])
	})
}

func TestContainerIsInjected(t *testing.T) {
	t.Run("is not injected", func(t *testing.T) {
		container := corev1.Container{}

		isInjected := ContainerIsInjected(container)

		require.False(t, isInjected)
	})
	t.Run("is injected", func(t *testing.T) {
		container := corev1.Container{
			VolumeMounts: []corev1.VolumeMount{
				{
					Name: workloadEnrichmentVolumeName,
				},
			},
		}

		isInjected := ContainerIsInjected(container)

		require.True(t, isInjected)
	})
}

func createTestMutationRequest(dk *dynakube.DynaKube, annotations map[string]string, withLabelSelector bool) *dtwebhook.MutationRequest {
	if dk == nil {
		dk = &dynakube.DynaKube{}
	}

	namespace := getTestNamespace()
	if withLabelSelector {
		namespace = getTestNamespaceWithMatchingLabel()
	}

	return dtwebhook.NewMutationRequest(
		context.Background(),
		*namespace,
		&corev1.Container{
			Name: dtwebhook.InstallContainerName,
		},
		getTestPod(annotations),
		*dk,
	)
}

func createTestReinvocationRequest(dk *dynakube.DynaKube, annotations map[string]string) *dtwebhook.ReinvocationRequest {
	request := createTestMutationRequest(dk, annotations, false).ToReinvocationRequest()
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
					Name:  "container-1",
					Image: "alpine-1",
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "volume",
							MountPath: "/volume",
						},
					},
				},
				{
					Name:  "container-2",
					Image: "alpine-2",
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

func getTestDynakube() *dynakube.DynaKube {
	return &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testDynakubeName,
			Namespace: testNamespaceName,
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL: testApiUrl,
			OneAgent: dynakube.OneAgentSpec{
				ApplicationMonitoring: &dynakube.ApplicationMonitoringSpec{},
			},
			ActiveGate: dynakube.ActiveGateSpec{
				Capabilities: []dynakube.CapabilityDisplayName{dynakube.MetricsIngestCapability.DisplayName},
			},
		},
		Status: dynakube.DynaKubeStatus{
			OneAgent: dynakube.OneAgentStatus{
				ConnectionInfoStatus: dynakube.OneAgentConnectionInfoStatus{
					ConnectionInfoStatus: dynakube.ConnectionInfoStatus{
						TenantUUID: "test-tenant",
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

func getTestNamespaceWithMatchingLabel() *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNamespaceName,
			Labels: map[string]string{
				dtwebhook.InjectionInstanceLabel: testDynakubeName,
				testLabelKeyMatching:             testLabelValue,
			},
		},
	}
}

package metadata

import (
	"testing"

	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSetInjectedAnnotation(t *testing.T) {
	t.Run("should add annotation to nil map", func(t *testing.T) {
		mut := NewMutator(nil)
		request := createTestMutationRequest(nil, nil)

		require.False(t, mut.IsInjected(request.BaseRequest))
		setInjectedAnnotation(request.Pod)
		require.Len(t, request.Pod.Annotations, 1)
		require.True(t, mut.IsInjected(request.BaseRequest))
	})
}

func TestWorkloadAnnotations(t *testing.T) {
	workloadInfoName := "workload-name"
	workloadInfoKind := "workload-kind"

	t.Run("should add annotation to nil map", func(t *testing.T) {
		request := createTestMutationRequest(nil, nil)

		require.Equal(t, "not-found", maputils.GetField(request.Pod.Annotations, AnnotationWorkloadName, "not-found"))
		setWorkloadAnnotations(request.Pod, &WorkloadInfo{Name: workloadInfoName, Kind: workloadInfoKind})
		require.Len(t, request.Pod.Annotations, 2)
		assert.Equal(t, workloadInfoName, maputils.GetField(request.Pod.Annotations, AnnotationWorkloadName, "not-found"))
		assert.Equal(t, workloadInfoKind, maputils.GetField(request.Pod.Annotations, AnnotationWorkloadKind, "not-found"))
	})
	t.Run("should lower case kind annotation", func(t *testing.T) {
		request := createTestMutationRequest(nil, nil)
		objectMeta := &metav1.PartialObjectMetadata{
			ObjectMeta: metav1.ObjectMeta{Name: workloadInfoName},
			TypeMeta:   metav1.TypeMeta{Kind: "SuperWorkload"},
		}

		setWorkloadAnnotations(request.Pod, newWorkloadInfo(objectMeta))
		assert.Contains(t, request.Pod.Annotations, AnnotationWorkloadKind)
		assert.Equal(t, "superworkload", request.Pod.Annotations[AnnotationWorkloadKind])
	})
}


// func TestAddPodAttributes(t *testing.T) {
// 	validateAttributes := func(t *testing.T, request dtwebhook.MutationRequest) podattr.Attributes {
// 		t.Helper()

// 		require.NotEmpty(t, request.InstallContainer.Args)

// 		rawArgs := []string{}

// 		for _, arg := range request.InstallContainer.Args {
// 			splitArg := strings.SplitN(arg, "=", 2)
// 			require.Len(t, splitArg, 2)
// 			rawArgs = append(rawArgs, splitArg[1])
// 		}

// 		attr, err := podattr.ParseAttributes(rawArgs)
// 		require.NoError(t, err)

// 		assert.Equal(t, request.DynaKube.Status.KubernetesClusterMEID, attr.DTClusterEntity)
// 		assert.Equal(t, request.DynaKube.Status.KubernetesClusterName, attr.ClusterName)
// 		assert.Equal(t, request.DynaKube.Status.KubeSystemUUID, attr.ClusterUID)
// 		assert.Contains(t, attr.PodName, consts.K8sPodNameEnv)
// 		assert.Contains(t, attr.PodUID, consts.K8sPodUIDEnv)
// 		assert.Contains(t, attr.NodeName, consts.K8sNodeNameEnv)
// 		assert.Equal(t, request.Pod.Namespace, attr.NamespaceName)

// 		require.Len(t, request.InstallContainer.Env, 3)
// 		assert.NotNil(t, env.FindEnvVar(request.InstallContainer.Env, consts.K8sPodNameEnv))
// 		assert.NotNil(t, env.FindEnvVar(request.InstallContainer.Env, consts.K8sPodUIDEnv))
// 		assert.NotNil(t, env.FindEnvVar(request.InstallContainer.Env, consts.K8sNodeNameEnv))

// 		return attr
// 	}

// 	validateAdditionAttributes := func(t *testing.T, request dtwebhook.MutationRequest) {
// 		t.Helper()

// 		attr := validateAttributes(t, request)

// 		require.NotEmpty(t, request.Pod.OwnerReferences)
// 		assert.Equal(t, strings.ToLower(request.Pod.OwnerReferences[0].Kind), attr.WorkloadKind)
// 		assert.Equal(t, request.Pod.OwnerReferences[0].Name, attr.WorkloadName)

// 		metaAnnotationCount := 0

// 		for key := range request.Namespace.Annotations {
// 			if strings.Contains(key, metacommon.AnnotationPrefix) {
// 				metaAnnotationCount++
// 			}
// 		}

// 		assert.Len(t, attr.UserDefined, metaAnnotationCount)
// 		require.Len(t, request.Pod.Annotations, 4+metaAnnotationCount)
// 		assert.Equal(t, strings.ToLower(request.Pod.OwnerReferences[0].Kind), request.Pod.Annotations[metacommon.AnnotationWorkloadKind])
// 		assert.Equal(t, request.Pod.OwnerReferences[0].Name, request.Pod.Annotations[metacommon.AnnotationWorkloadName])
// 		assert.Equal(t, "true", request.Pod.Annotations[metacommon.AnnotationInjected])
// 	}

// 	t.Run("metadata enrichment passes => additional args and annotations", func(t *testing.T) {
// 		initContainer := corev1.Container{
// 			Args: []string{},
// 		}
// 		pod := corev1.Pod{
// 			ObjectMeta: metav1.ObjectMeta{
// 				Namespace: "test",
// 				OwnerReferences: []metav1.OwnerReference{
// 					{
// 						Name:       "owner",
// 						APIVersion: "v1",
// 						Kind:       "ReplicationController",
// 						Controller: ptr.To(true),
// 					},
// 				},
// 			},
// 		}
// 		owner := corev1.ReplicationController{
// 			TypeMeta: metav1.TypeMeta{
// 				APIVersion: "v1",
// 				Kind:       "ReplicationController",
// 			},
// 			ObjectMeta: metav1.ObjectMeta{
// 				Name: "owner",
// 			},
// 		}

// 		expectedPod := pod.DeepCopy()

// 		request := dtwebhook.MutationRequest{
// 			BaseRequest: &dtwebhook.BaseRequest{
// 				Pod: &pod,
// 				DynaKube: dynakube.DynaKube{
// 					Spec: dynakube.DynaKubeSpec{
// 						MetadataEnrichment: dynakube.MetadataEnrichment{
// 							Enabled: ptr.To(true),
// 						},
// 					},
// 					Status: dynakube.DynaKubeStatus{
// 						KubernetesClusterMEID: "meid",
// 						KubeSystemUUID:        "systemuuid",
// 						KubernetesClusterName: "meidname",
// 					},
// 				},
// 			},
// 			InstallContainer: &initContainer,
// 		}

// 		err := addPodAttributes(&request)
// 		require.NoError(t, err)
// 		require.NotEqual(t, *expectedPod, *request.Pod)

// 		validateAttributes(t, request)
// 	})

// 	t.Run("metadata enrichment fails => error", func(t *testing.T) {
// 		injector := createTestWebhookBase()
// 		injector.metaClient = fake.NewClient()

// 		initContainer := corev1.Container{
// 			Args: []string{},
// 		}
// 		pod := corev1.Pod{
// 			ObjectMeta: metav1.ObjectMeta{
// 				Namespace: "test",
// 				OwnerReferences: []metav1.OwnerReference{
// 					{
// 						Name:       "owner",
// 						APIVersion: "v1",
// 						Kind:       "ReplicationController",
// 						Controller: ptr.To(true),
// 					},
// 				},
// 			},
// 		}

// 		expectedPod := pod.DeepCopy()

// 		request := dtwebhook.MutationRequest{
// 			BaseRequest: &dtwebhook.BaseRequest{
// 				Pod: &pod,
// 				Namespace: corev1.Namespace{
// 					ObjectMeta: metav1.ObjectMeta{
// 						Annotations: map[string]string{
// 							metacommon.AnnotationPrefix + "/test": "test",
// 						},
// 					},
// 				},
// 				DynaKube: dynakube.DynaKube{
// 					Spec: dynakube.DynaKubeSpec{
// 						MetadataEnrichment: dynakube.MetadataEnrichment{
// 							Enabled: ptr.To(true),
// 						},
// 					},
// 					Status: dynakube.DynaKubeStatus{
// 						KubernetesClusterMEID: "meid",
// 						KubeSystemUUID:        "systemuuid",
// 						KubernetesClusterName: "meidname",
// 					},
// 				},
// 			},
// 			InstallContainer: &initContainer,
// 		}

// 		err := addPodAttributes(&request)
// 		require.Error(t, err)
// 		require.Equal(t, *expectedPod, *request.Pod)
// 	})
// }

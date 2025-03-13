package v2

import (
	"encoding/json"
	"strings"
	"testing"

	containerattr "github.com/Dynatrace/dynatrace-bootstrapper/pkg/configure/attributes/container"
	podattr "github.com/Dynatrace/dynatrace-bootstrapper/pkg/configure/attributes/pod"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	metacommon "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/common/metadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/v2/common/volumes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestAddPodAttributes(t *testing.T) {
	validateAttributes := func(t *testing.T, request dtwebhook.MutationRequest) podattr.Attributes {
		t.Helper()

		require.NotEmpty(t, request.InstallContainer.Args)

		rawArgs := []string{}

		for _, arg := range request.InstallContainer.Args {
			splitArg := strings.SplitN(arg, "=", 2)
			require.Len(t, splitArg, 2)
			rawArgs = append(rawArgs, splitArg[1])
		}

		attr, err := podattr.ParseAttributes(rawArgs)
		require.NoError(t, err)

		assert.Equal(t, request.DynaKube.Status.KubernetesClusterMEID, attr.DTClusterEntity)
		assert.Equal(t, request.DynaKube.Status.KubeSystemUUID, attr.ClusterUId)
		assert.Contains(t, attr.PodName, consts.K8sPodNameEnv)
		assert.Contains(t, attr.PodUid, consts.K8sPodUIDEnv)
		assert.Equal(t, request.Pod.Namespace, attr.NamespaceName)

		require.Len(t, request.InstallContainer.Env, 2)
		assert.NotNil(t, env.FindEnvVar(request.InstallContainer.Env, consts.K8sPodNameEnv))
		assert.NotNil(t, env.FindEnvVar(request.InstallContainer.Env, consts.K8sPodUIDEnv))

		return attr
	}

	validateAdditionAttributes := func(t *testing.T, request dtwebhook.MutationRequest) {
		t.Helper()

		attr := validateAttributes(t, request)

		require.NotEmpty(t, request.Pod.OwnerReferences)
		assert.Equal(t, strings.ToLower(request.Pod.OwnerReferences[0].Kind), attr.WorkloadKind)
		assert.Equal(t, request.Pod.OwnerReferences[0].Name, attr.WorkloadName)

		metaAnnotationCount := 0

		for key := range request.Namespace.Annotations {
			if strings.Contains(key, metacommon.AnnotationPrefix) {
				metaAnnotationCount++
			}
		}

		assert.Len(t, attr.UserDefined, metaAnnotationCount)
		require.Len(t, request.Pod.Annotations, 3+metaAnnotationCount)
		assert.Equal(t, strings.ToLower(request.Pod.OwnerReferences[0].Kind), request.Pod.Annotations[metacommon.AnnotationWorkloadKind])
		assert.Equal(t, request.Pod.OwnerReferences[0].Name, request.Pod.Annotations[metacommon.AnnotationWorkloadName])
		assert.Equal(t, "true", request.Pod.Annotations[metacommon.AnnotationInjected])
	}

	t.Run("add attributes and related envs, do not change pod or app-container", func(t *testing.T) {
		injector := createTestInjectorBase()

		initContainer := corev1.Container{
			Args: []string{},
		}
		pod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "test",
			},
		}

		expectedPod := pod.DeepCopy()

		request := dtwebhook.MutationRequest{
			BaseRequest: &dtwebhook.BaseRequest{
				Pod: &pod,
				DynaKube: dynakube.DynaKube{
					Status: dynakube.DynaKubeStatus{
						KubernetesClusterMEID: "meid",
						KubeSystemUUID:        "systemuuid",
					},
				},
			},
			InstallContainer: &initContainer,
		}

		err := injector.addPodAttributes(&request)
		require.NoError(t, err)

		require.Equal(t, *expectedPod, *request.BaseRequest.Pod)
		validateAttributes(t, request)
	})

	t.Run("metadata enrichment passes => additional args and annotations", func(t *testing.T) {
		initContainer := corev1.Container{
			Args: []string{},
		}
		pod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "test",
				OwnerReferences: []metav1.OwnerReference{
					{
						Name:       "owner",
						APIVersion: "v1",
						Kind:       "ReplicationController",
						Controller: ptr.To(true),
					},
				},
			},
		}
		owner := corev1.ReplicationController{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "ReplicationController",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "owner",
			},
		}
		injector := createTestInjectorBase()
		injector.metaClient = fake.NewClient(&owner, &pod)

		expectedPod := pod.DeepCopy()

		request := dtwebhook.MutationRequest{
			BaseRequest: &dtwebhook.BaseRequest{
				Pod: &pod,
				DynaKube: dynakube.DynaKube{
					Spec: dynakube.DynaKubeSpec{
						MetadataEnrichment: dynakube.MetadataEnrichment{
							Enabled: ptr.To(true),
						},
					},
				},
			},
			InstallContainer: &initContainer,
		}

		err := injector.addPodAttributes(&request)
		require.NoError(t, err)
		require.NotEqual(t, *expectedPod, *request.BaseRequest.Pod)

		validateAdditionAttributes(t, request)
	})

	t.Run("metadata enrichment fails => error", func(t *testing.T) {
		injector := createTestInjectorBase()
		injector.metaClient = fake.NewClient()

		initContainer := corev1.Container{
			Args: []string{},
		}
		pod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "test",
				OwnerReferences: []metav1.OwnerReference{
					{
						Name:       "owner",
						APIVersion: "v1",
						Kind:       "ReplicationController",
						Controller: ptr.To(true),
					},
				},
			},
		}

		expectedPod := pod.DeepCopy()

		request := dtwebhook.MutationRequest{
			BaseRequest: &dtwebhook.BaseRequest{
				Pod: &pod,
				Namespace: corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							metacommon.AnnotationPrefix + "/test": "test",
						},
					},
				},
				DynaKube: dynakube.DynaKube{
					Spec: dynakube.DynaKubeSpec{
						MetadataEnrichment: dynakube.MetadataEnrichment{
							Enabled: ptr.To(true),
						},
					},
					Status: dynakube.DynaKubeStatus{
						KubernetesClusterMEID: "meid",
						KubeSystemUUID:        "systemuuid",
					},
				},
			},
			InstallContainer: &initContainer,
		}

		err := injector.addPodAttributes(&request)
		require.Error(t, err)
		require.Equal(t, *expectedPod, *request.BaseRequest.Pod)
	})
}

func TestAddContainerAttributes(t *testing.T) {
	validateContainerAttributes := func(t *testing.T, pod corev1.Pod, args []string) {
		t.Helper()

		require.NotEmpty(t, args)

		for _, arg := range args {
			splitArg := strings.Split(arg, "=")
			require.Len(t, splitArg, 2)

			var attr containerattr.Attributes

			require.NoError(t, json.Unmarshal([]byte(splitArg[1]), &attr))
			assert.Contains(t, pod.Spec.Containers, corev1.Container{
				Name:  attr.ContainerName,
				Image: attr.ToURI(),
			})
		}
	}

	t.Run("add container-attributes, do not change pod or app-container", func(t *testing.T) {
		app1Container := corev1.Container{
			Name:  "app-1-name",
			Image: "registry1.example.com/repository/image:tag",
		}
		app2Container := corev1.Container{
			Name:  "app-2-name",
			Image: "registry2.example.com/repository/image:tag",
		}
		initContainer := corev1.Container{
			Args: []string{},
		}
		pod := corev1.Pod{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					app1Container,
					app2Container,
				},
			},
		}

		expectedPod := pod.DeepCopy()

		request := dtwebhook.MutationRequest{
			BaseRequest: &dtwebhook.BaseRequest{
				Pod: &pod,
			},
			InstallContainer: &initContainer,
		}

		addContainerAttributes(&request)

		require.Equal(t, *expectedPod, *request.BaseRequest.Pod)

		validateContainerAttributes(t, pod, initContainer.Args)
	})

	t.Run("no new container ==> no new arg", func(t *testing.T) {
		app1Container := corev1.Container{
			Name:  "app-1-name",
			Image: "registry1.example.com/repository/image:tag",
		}
		volumes.AddConfigVolumeMount(&app1Container)

		app2Container := corev1.Container{
			Name:  "app-2-name",
			Image: "registry2.example.com/repository/image:tag",
		}
		volumes.AddConfigVolumeMount(&app2Container)

		initContainer := corev1.Container{
			Args: []string{},
		}
		pod := corev1.Pod{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					app1Container,
					app2Container,
				},
			},
		}

		expectedPod := pod.DeepCopy()

		request := dtwebhook.MutationRequest{
			BaseRequest: &dtwebhook.BaseRequest{
				Pod: &pod,
			},
			InstallContainer: &initContainer,
		}

		addContainerAttributes(&request)

		require.Equal(t, *expectedPod, *request.BaseRequest.Pod)
		require.Empty(t, initContainer.Args)
	})

	t.Run("partially new => only add new", func(t *testing.T) {
		app1Container := corev1.Container{
			Name:  "app-1-name",
			Image: "registry1.example.com/repository/image:tag",
		}
		volumes.AddConfigVolumeMount(&app1Container)

		app2Container := corev1.Container{
			Name:  "app-2-name",
			Image: "registry2.example.com/repository/image:tag",
		}

		initContainer := corev1.Container{
			Args: []string{},
		}
		pod := corev1.Pod{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					app1Container,
					app2Container,
				},
			},
		}

		expectedPod := pod.DeepCopy()

		request := dtwebhook.MutationRequest{
			BaseRequest: &dtwebhook.BaseRequest{
				Pod: &pod,
			},
			InstallContainer: &initContainer,
		}

		addContainerAttributes(&request)

		require.Equal(t, *expectedPod, *request.BaseRequest.Pod)
		require.Len(t, initContainer.Args, 1)
		validateContainerAttributes(t, pod, initContainer.Args)
	})
}

func TestCreateImageInfo(t *testing.T) {
	t.Run("with tag", func(t *testing.T) {
		imageURI := "registry.example.com/repository/image:tag"

		imageInfo := createImageInfo(imageURI)

		require.Equal(t, containerattr.ImageInfo{
			Registry:    "registry.example.com",
			Repository:  "repository/image",
			Tag:         "tag",
			ImageDigest: "",
		}, imageInfo)
	})
	t.Run("with digest", func(t *testing.T) {
		imageURI := "registry.example.com/repository/image@sha256:7173b809ca12ec5dee4506cd86be934c4596dd234ee82c0662eac04a8c2c71dc"

		imageInfo := createImageInfo(imageURI)

		require.Equal(t, containerattr.ImageInfo{
			Registry:    "registry.example.com",
			Repository:  "repository/image",
			Tag:         "",
			ImageDigest: "sha256:7173b809ca12ec5dee4506cd86be934c4596dd234ee82c0662eac04a8c2c71dc",
		}, imageInfo)
	})
	t.Run("with digest and tag", func(t *testing.T) {
		imageURI := "registry.example.com/repository/image:tag@sha256:7173b809ca12ec5dee4506cd86be934c4596dd234ee82c0662eac04a8c2c71dc"

		imageInfo := createImageInfo(imageURI)

		require.Equal(t, containerattr.ImageInfo{
			Registry:    "registry.example.com",
			Repository:  "repository/image",
			Tag:         "tag",
			ImageDigest: "sha256:7173b809ca12ec5dee4506cd86be934c4596dd234ee82c0662eac04a8c2c71dc",
		}, imageInfo)
	})
	t.Run("with missing tag", func(t *testing.T) {
		imageURI := "registry.example.com/repository/image"

		imageInfo := createImageInfo(imageURI)

		require.Equal(t, containerattr.ImageInfo{
			Registry:    "registry.example.com",
			Repository:  "repository/image",
			Tag:         "",
			ImageDigest: "",
		}, imageInfo)
	})

	t.Run("actual example", func(t *testing.T) {
		imageURI := "docker.io/php:fpm-stretch"

		imageInfo := createImageInfo(imageURI)

		require.Equal(t, containerattr.ImageInfo{
			Registry:    "docker.io",
			Repository:  "php",
			Tag:         "fpm-stretch",
			ImageDigest: "",
		}, imageInfo)
	})
}

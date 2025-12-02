package injection

import (
	"encoding/json"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	containerattr "github.com/Dynatrace/dynatrace-bootstrapper/cmd/configure/attributes/container"
	podattr "github.com/Dynatrace/dynatrace-bootstrapper/cmd/configure/attributes/pod"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/volumes"
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
			_, rawArg, found := strings.Cut(arg, "=")
			require.True(t, found, "missing '=': "+arg)
			rawArgs = append(rawArgs, rawArg)
		}

		attr, err := podattr.ParseAttributes(rawArgs)
		require.NoError(t, err)

		assert.Equal(t, request.DynaKube.Status.KubernetesClusterMEID, attr.DTClusterEntity)
		assert.Equal(t, request.DynaKube.Status.KubernetesClusterName, attr.ClusterName)
		assert.Equal(t, request.DynaKube.Status.KubeSystemUUID, attr.ClusterUID)
		assert.Contains(t, attr.PodName, K8sPodNameEnv)
		assert.Contains(t, attr.PodUID, K8sPodUIDEnv)
		assert.Contains(t, attr.NodeName, K8sNodeNameEnv)
		assert.Equal(t, request.Pod.Namespace, attr.NamespaceName)

		assertDeprecatedAttributes(t, attr)

		require.Len(t, request.InstallContainer.Env, 3)
		assert.NotNil(t, k8senv.Find(request.InstallContainer.Env, K8sPodNameEnv))
		assert.NotNil(t, k8senv.Find(request.InstallContainer.Env, K8sPodUIDEnv))
		assert.NotNil(t, k8senv.Find(request.InstallContainer.Env, K8sNodeNameEnv))

		return attr
	}

	t.Run("args and envs added", func(t *testing.T) {
		initContainer := corev1.Container{
			Args: []string{},
		}
		pod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "test"},
		}

		expectedPod := pod.DeepCopy()

		request := dtwebhook.MutationRequest{
			BaseRequest: &dtwebhook.BaseRequest{
				Pod: &pod,
				DynaKube: dynakube.DynaKube{
					Spec: dynakube.DynaKubeSpec{
						MetadataEnrichment: metadataenrichment.Spec{
							Enabled: ptr.To(true),
						},
					},
					Status: dynakube.DynaKubeStatus{
						KubernetesClusterMEID: "meid",
						KubeSystemUUID:        "systemuuid",
						KubernetesClusterName: "meidname",
					},
				},
			},
			InstallContainer: &initContainer,
		}

		err := addPodAttributes(&request)
		require.NoError(t, err)
		require.Equal(t, *expectedPod, *request.Pod)

		validateAttributes(t, request)
	})
}

func TestAddContainerAttributes(t *testing.T) {
	// request to pre-mount required volumes: OneAgent or Enrichment or both
	vmBaseRequest := &dtwebhook.BaseRequest{
		Pod: &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					dtwebhook.AnnotationInjectionSplitMounts: "false",
				},
			},
		},
		DynaKube: dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				MetadataEnrichment: metadataenrichment.Spec{
					Enabled: ptr.To(true),
				},
				OneAgent: oneagent.Spec{
					ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{},
				},
			},
		},
	}

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
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      volumes.ConfigVolumeName,
						MountPath: volumes.ConfigMountPath,
						SubPath:   attr.ContainerName,
					},
				},
			})
		}
	}

	t.Run("add container-attributes + mount", func(t *testing.T) {
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

		request := dtwebhook.MutationRequest{
			BaseRequest: &dtwebhook.BaseRequest{
				Pod: &pod,
			},
			InstallContainer: &initContainer,
		}

		addContainerAttributes(&request)

		validateContainerAttributes(t, pod, initContainer.Args)
	})

	t.Run("no new container ==> no new arg", func(t *testing.T) {
		app1Container := corev1.Container{
			Name:  "app-1-name",
			Image: "registry1.example.com/repository/image:tag",
		}
		volumes.AddConfigVolumeMount(&app1Container, vmBaseRequest)

		app2Container := corev1.Container{
			Name:  "app-2-name",
			Image: "registry2.example.com/repository/image:tag",
		}
		volumes.AddConfigVolumeMount(&app2Container, vmBaseRequest)

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

		request := dtwebhook.MutationRequest{
			BaseRequest: &dtwebhook.BaseRequest{
				Pod: &pod,
			},
			InstallContainer: &initContainer,
		}

		addContainerAttributes(&request)

		require.Empty(t, initContainer.Args)
	})

	t.Run("partially new => only add new", func(t *testing.T) {
		app1Container := corev1.Container{
			Name:  "app-1-name",
			Image: "registry1.example.com/repository/image:tag",
		}
		volumes.AddConfigVolumeMount(&app1Container, vmBaseRequest)

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

		request := dtwebhook.MutationRequest{
			BaseRequest: &dtwebhook.BaseRequest{
				Pod: &pod,
			},
			InstallContainer: &initContainer,
		}

		addContainerAttributes(&request)

		require.Len(t, initContainer.Args, 1)
		validateContainerAttributes(t, pod, initContainer.Args)
	})
}

func TestAddContainerAttributesWithSplitVolumes(t *testing.T) {
	// request to pre-mount required volumes: OneAgent or Enrichment or both
	vmBaseRequest := func(metadataEnrichment bool, oneAgent bool) *dtwebhook.BaseRequest {
		br := &dtwebhook.BaseRequest{
			Pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						dtwebhook.AnnotationInjectionSplitMounts: "true",
					},
				},
			},
			DynaKube: dynakube.DynaKube{
				Spec: dynakube.DynaKubeSpec{
					MetadataEnrichment: metadataenrichment.Spec{
						Enabled: ptr.To(metadataEnrichment),
					},
				},
			},
		}
		if oneAgent {
			br.DynaKube.Spec.OneAgent = oneagent.Spec{
				ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{},
			}
		}

		return br
	}

	validateContainerAttributes := func(t *testing.T, pod corev1.Pod, args []string) {
		t.Helper()

		require.NotEmpty(t, args)

		for i := range pod.Spec.Containers {
			slices.SortFunc(pod.Spec.Containers[i].VolumeMounts, func(a, b corev1.VolumeMount) int {
				return strings.Compare(a.MountPath, b.MountPath)
			})
		}

		for _, arg := range args {
			splitArg := strings.Split(arg, "=")
			require.Len(t, splitArg, 2)

			var attr containerattr.Attributes

			require.NoError(t, json.Unmarshal([]byte(splitArg[1]), &attr))

			assert.Contains(t, pod.Spec.Containers, corev1.Container{
				Name:  attr.ContainerName,
				Image: attr.ToURI(),
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      volumes.ConfigVolumeName,
						MountPath: filepath.Join(volumes.ConfigMountPath, "enrichment", "dt_metadata.json"),
						SubPath:   filepath.Join(attr.ContainerName, "enrichment", "dt_metadata.json"),
					},
					{
						Name:      volumes.ConfigVolumeName,
						MountPath: filepath.Join(volumes.ConfigMountPath, "enrichment", "dt_metadata.properties"),
						SubPath:   filepath.Join(attr.ContainerName, "enrichment", "dt_metadata.properties"),
					},
					{
						Name:      volumes.ConfigVolumeName,
						MountPath: filepath.Join(volumes.ConfigMountPath, "enrichment", "endpoints"),
						SubPath:   filepath.Join(attr.ContainerName, "enrichment", "endpoints"),
					},
					{
						Name:      volumes.ConfigVolumeName,
						MountPath: filepath.Join(volumes.ConfigMountPath, "oneagent"),
						SubPath:   filepath.Join(attr.ContainerName, "oneagent"),
					},
				},
			})
		}
	}

	validateContainerAttributesforOneAgentInjection := func(t *testing.T, pod corev1.Pod, args []string) {
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
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      volumes.ConfigVolumeName,
						MountPath: filepath.Join(volumes.ConfigMountPath, "oneagent"),
						SubPath:   filepath.Join(attr.ContainerName, "oneagent"),
					},
				},
			})
		}
	}

	validateContainerAttributesforMetadataEnrichment := func(t *testing.T, pod corev1.Pod, args []string) {
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
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      volumes.ConfigVolumeName,
						MountPath: filepath.Join(volumes.ConfigMountPath, "enrichment", "dt_metadata.json"),
						SubPath:   filepath.Join(attr.ContainerName, "enrichment", "dt_metadata.json"),
					},
					{
						Name:      volumes.ConfigVolumeName,
						MountPath: filepath.Join(volumes.ConfigMountPath, "enrichment", "dt_metadata.properties"),
						SubPath:   filepath.Join(attr.ContainerName, "enrichment", "dt_metadata.properties"),
					},
					{
						Name:      volumes.ConfigVolumeName,
						MountPath: filepath.Join(volumes.ConfigMountPath, "enrichment", "endpoints"),
						SubPath:   filepath.Join(attr.ContainerName, "enrichment", "endpoints"),
					},
				},
			})
		}
	}

	t.Run("add container-attributes + mount", func(t *testing.T) {
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
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					dtwebhook.AnnotationInjectionSplitMounts: "true",
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					app1Container,
					app2Container,
				},
			},
		}

		request := dtwebhook.MutationRequest{
			BaseRequest: &dtwebhook.BaseRequest{
				Pod: &pod,
				DynaKube: dynakube.DynaKube{
					Spec: dynakube.DynaKubeSpec{
						MetadataEnrichment: metadataenrichment.Spec{
							Enabled: ptr.To(true),
						},
						OneAgent: oneagent.Spec{
							ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{},
						},
					},
				},
			},
			InstallContainer: &initContainer,
		}

		_, err := addContainerAttributes(&request)
		require.NoError(t, err)

		validateContainerAttributes(t, pod, initContainer.Args)
	})

	t.Run("no new container ==> no new arg", func(t *testing.T) {
		app1Container := corev1.Container{
			Name:  "app-1-name",
			Image: "registry1.example.com/repository/image:tag",
		}
		volumes.AddConfigVolumeMount(&app1Container, vmBaseRequest(true, true))

		app2Container := corev1.Container{
			Name:  "app-2-name",
			Image: "registry2.example.com/repository/image:tag",
		}
		volumes.AddConfigVolumeMount(&app2Container, vmBaseRequest(true, true))

		initContainer := corev1.Container{
			Args: []string{},
		}
		pod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					dtwebhook.AnnotationInjectionSplitMounts: "true",
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					app1Container,
					app2Container,
				},
			},
		}

		request := dtwebhook.MutationRequest{
			BaseRequest: &dtwebhook.BaseRequest{
				Pod: &pod,
				DynaKube: dynakube.DynaKube{
					Spec: dynakube.DynaKubeSpec{
						MetadataEnrichment: metadataenrichment.Spec{
							Enabled: ptr.To(true),
						},
						OneAgent: oneagent.Spec{
							ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{},
						},
					},
				},
			},
			InstallContainer: &initContainer,
		}

		_, err := addContainerAttributes(&request)
		require.NoError(t, err)

		require.Empty(t, initContainer.Args)
	})

	t.Run("partially new => only add new", func(t *testing.T) {
		app1Container := corev1.Container{
			Name:  "app-1-name",
			Image: "registry1.example.com/repository/image:tag",
		}
		volumes.AddConfigVolumeMount(&app1Container, vmBaseRequest(true, true))

		app2Container := corev1.Container{
			Name:  "app-2-name",
			Image: "registry2.example.com/repository/image:tag",
		}

		initContainer := corev1.Container{
			Args: []string{},
		}
		pod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					dtwebhook.AnnotationInjectionSplitMounts: "true",
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					app1Container,
					app2Container,
				},
			},
		}

		request := dtwebhook.MutationRequest{
			BaseRequest: &dtwebhook.BaseRequest{
				Pod: &pod,
				DynaKube: dynakube.DynaKube{
					Spec: dynakube.DynaKubeSpec{
						MetadataEnrichment: metadataenrichment.Spec{
							Enabled: ptr.To(true),
						},
						OneAgent: oneagent.Spec{
							ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{},
						},
					},
				},
			},
			InstallContainer: &initContainer,
		}

		_, err := addContainerAttributes(&request)
		require.NoError(t, err)

		require.Len(t, initContainer.Args, 1)
		validateContainerAttributes(t, pod, initContainer.Args)
	})

	t.Run("partially new => add oneagent or enrichment", func(t *testing.T) {
		app1Container := corev1.Container{
			Name:  "app-1-name",
			Image: "registry1.example.com/repository/image:tag",
		}
		volumes.AddConfigVolumeMount(&app1Container, vmBaseRequest(false, true))

		app2Container := corev1.Container{
			Name:  "app-2-name",
			Image: "registry2.example.com/repository/image:tag",
		}
		volumes.AddConfigVolumeMount(&app2Container, vmBaseRequest(true, false))

		initContainer := corev1.Container{
			Args: []string{},
		}
		pod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					dtwebhook.AnnotationInjectionSplitMounts: "true",
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					app1Container,
					app2Container,
				},
			},
		}

		request := dtwebhook.MutationRequest{
			BaseRequest: &dtwebhook.BaseRequest{
				Pod: &pod,
				DynaKube: dynakube.DynaKube{
					Spec: dynakube.DynaKubeSpec{
						MetadataEnrichment: metadataenrichment.Spec{
							Enabled: ptr.To(true),
						},
						OneAgent: oneagent.Spec{
							ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{},
						},
					},
				},
			},
			InstallContainer: &initContainer,
		}

		_, err := addContainerAttributes(&request)
		require.NoError(t, err)

		require.Len(t, initContainer.Args, 2)
		validateContainerAttributes(t, pod, initContainer.Args)
	})

	t.Run("partially new => add oneagent", func(t *testing.T) {
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
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					dtwebhook.AnnotationInjectionSplitMounts: "true",
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					app1Container,
					app2Container,
				},
			},
		}

		request := dtwebhook.MutationRequest{
			BaseRequest: &dtwebhook.BaseRequest{
				Pod: &pod,
				DynaKube: dynakube.DynaKube{
					Spec: dynakube.DynaKubeSpec{
						MetadataEnrichment: metadataenrichment.Spec{
							Enabled: ptr.To(false),
						},
						OneAgent: oneagent.Spec{
							ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{},
						},
					},
				},
			},
			InstallContainer: &initContainer,
		}

		_, err := addContainerAttributes(&request)
		require.NoError(t, err)

		require.Len(t, initContainer.Args, 2)
		validateContainerAttributesforOneAgentInjection(t, pod, initContainer.Args)
	})

	t.Run("partially new => add enrichment", func(t *testing.T) {
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
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					dtwebhook.AnnotationInjectionSplitMounts: "true",
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					app1Container,
					app2Container,
				},
			},
		}

		request := dtwebhook.MutationRequest{
			BaseRequest: &dtwebhook.BaseRequest{
				Pod: &pod,
				DynaKube: dynakube.DynaKube{
					Spec: dynakube.DynaKubeSpec{
						MetadataEnrichment: metadataenrichment.Spec{
							Enabled: ptr.To(true),
						},
					},
				},
			},
			InstallContainer: &initContainer,
		}

		_, err := addContainerAttributes(&request)
		require.NoError(t, err)

		require.Len(t, initContainer.Args, 2)
		validateContainerAttributesforMetadataEnrichment(t, pod, initContainer.Args)
	})
}

func TestCreateImageInfo(t *testing.T) {
	type testCase struct {
		title string
		in    string
		out   containerattr.ImageInfo
	}

	testCases := []testCase{
		{
			title: "empty URI",
			in:    "",
			out:   containerattr.ImageInfo{},
		},
		{
			title: "URI with tag",
			in:    "registry.example.com/repository/image:tag",
			out: containerattr.ImageInfo{
				Registry:    "registry.example.com",
				Repository:  "repository/image",
				Tag:         "tag",
				ImageDigest: "",
			},
		},
		{
			title: "URI with digest",
			in:    "registry.example.com/repository/image@sha256:7173b809ca12ec5dee4506cd86be934c4596dd234ee82c0662eac04a8c2c71dc",
			out: containerattr.ImageInfo{
				Registry:    "registry.example.com",
				Repository:  "repository/image",
				Tag:         "",
				ImageDigest: "sha256:7173b809ca12ec5dee4506cd86be934c4596dd234ee82c0662eac04a8c2c71dc",
			},
		},
		{
			title: "URI with digest and tag",
			in:    "registry.example.com/repository/image:tag@sha256:7173b809ca12ec5dee4506cd86be934c4596dd234ee82c0662eac04a8c2c71dc",
			out: containerattr.ImageInfo{
				Registry:    "registry.example.com",
				Repository:  "repository/image",
				Tag:         "tag",
				ImageDigest: "sha256:7173b809ca12ec5dee4506cd86be934c4596dd234ee82c0662eac04a8c2c71dc",
			},
		},
		{
			title: "URI with missing tag",
			in:    "registry.example.com/repository/image",
			out: containerattr.ImageInfo{
				Registry:   "registry.example.com",
				Repository: "repository/image",
			},
		},
		{
			title: "URI with docker.io (special case in certain libraries)",
			in:    "docker.io/php:fpm-stretch",
			out: containerattr.ImageInfo{
				Registry:   "docker.io",
				Repository: "php",
				Tag:        "fpm-stretch",
			},
		},
		{
			title: "URI with missing registry",
			in:    "php:fpm-stretch",
			out: containerattr.ImageInfo{
				Repository: "php",
				Tag:        "fpm-stretch",
			},
		},
	}
	for _, test := range testCases {
		t.Run(test.title, func(t *testing.T) {
			imageInfo := createImageInfo(test.in)

			require.Equal(t, test.out, imageInfo)
		})
	}
}

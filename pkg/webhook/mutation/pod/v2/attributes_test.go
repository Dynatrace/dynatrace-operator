package v2

import (
	"encoding/json"
	"strings"
	"testing"

	containerattr "github.com/Dynatrace/dynatrace-bootstrapper/pkg/configure/attributes/container"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/v2/common/volumes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestAddPodAttributes(t *testing.T) {

}

func TestAddContainerAttributes(t *testing.T) {
	validateContainerArgs := func(t *testing.T, pod corev1.Pod, args []string) {
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

	t.Run("add attributes, do not change pod or app-container", func(t *testing.T) {
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

		validateContainerArgs(t, pod, initContainer.Args)
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
		validateContainerArgs(t, pod, initContainer.Args)
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

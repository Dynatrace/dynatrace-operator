package k8scontainer

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	containerName = "test-name"
	doNotFindName = "do-not-find"
	podName       = "test-pod"
)

var podSpec = corev1.PodSpec{
	Containers: []corev1.Container{
		{
			Name: containerName,
		},
	},
}

var pod = corev1.Pod{
	ObjectMeta: metav1.ObjectMeta{
		Name: podName,
	},
	Spec: podSpec,
}

func TestFindContainerInPod(t *testing.T) {
	t.Run("container is found in pod", func(t *testing.T) {
		containerInPod, err := FindInPod(pod, containerName)
		require.NoError(t, err)
		require.NotNil(t, containerInPod)
	})
	t.Run("container is not found in pod", func(t *testing.T) {
		containerInPod, err := FindInPod(pod, doNotFindName)
		require.Error(t, err)
		require.Nil(t, containerInPod)
	})
}

func TestFindContainerInPodSpec(t *testing.T) {
	t.Run("container is found in podSpec", func(t *testing.T) {
		containerInPodSpec := FindInPodSpec(&podSpec, containerName)
		require.NotNil(t, containerInPodSpec)
	})
	t.Run("container is not found in podSpec", func(t *testing.T) {
		containerInPodSpec := FindInPodSpec(&podSpec, doNotFindName)
		require.Nil(t, containerInPodSpec)
	})
}

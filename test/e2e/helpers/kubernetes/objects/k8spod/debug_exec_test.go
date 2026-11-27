// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package k8spod

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
	"k8s.io/utils/ptr"
)

const (
	testNamespace = "dynatrace"
	testPod       = "csi-driver"
)

func TestDebugVolumeMounts(t *testing.T) {
	target := []corev1.VolumeMount{
		{Name: "data-dir", MountPath: "/data", MountPropagation: ptr.To(corev1.MountPropagationBidirectional)},
		{Name: "config", MountPath: "/etc/config", ReadOnly: true},
		{Name: "sub", MountPath: "/sub", SubPath: "x"},
		{Name: "sub-expr", MountPath: "/sub-expr", SubPathExpr: "$(POD_NAME)"},
		{Name: "kube-api-access", MountPath: serviceAccountMountPath, ReadOnly: true},
	}

	mounts := debugVolumeMounts(target)

	require.Len(t, mounts, 2)
	assert.Equal(t, "/data", mounts[0].MountPath)
	assert.Equal(t, corev1.MountPropagationBidirectional, *mounts[0].MountPropagation, "the debug container has the privileges of the target")
	assert.Equal(t, "/etc/config", mounts[1].MountPath)
	assert.True(t, mounts[1].ReadOnly)
}

func TestNewDebugContainer(t *testing.T) {
	target := corev1.Container{
		Name: "server",
		SecurityContext: &corev1.SecurityContext{
			Privileged:     new(true),
			RunAsUser:      new(int64(0)),
			SELinuxOptions: &corev1.SELinuxOptions{Level: "s0"},
		},
		LivenessProbe: &corev1.Probe{},
		VolumeMounts:  []corev1.VolumeMount{{Name: "data-dir", MountPath: "/data"}},
	}

	container := newDebugContainer("e2e-debug-server-abcde", target)

	assert.Equal(t, "e2e-debug-server-abcde", container.Name)
	assert.Empty(t, container.TargetContainerName)
	assert.Equal(t, defaultDebugImage, container.Image)
	assert.Equal(t, target.SecurityContext, container.SecurityContext)
	assert.NotSame(t, target.SecurityContext, container.SecurityContext, "must be a copy")
	assert.Nil(t, container.LivenessProbe)
	assert.Len(t, container.VolumeMounts, 1)

	t.Run("without a security context", func(t *testing.T) {
		assert.Nil(t, newDebugContainer("x", corev1.Container{}).SecurityContext)
	})
	t.Run("image can be overridden", func(t *testing.T) {
		t.Setenv(DebugImageEnv, "example.com/shell:1")
		assert.Equal(t, "example.com/shell:1", newDebugContainer("x", corev1.Container{}).Image)
	})
}

func TestFindUsableDebugContainer(t *testing.T) {
	ephemeral := func(names ...string) []corev1.EphemeralContainer {
		var containers []corev1.EphemeralContainer
		for _, name := range names {
			containers = append(containers, corev1.EphemeralContainer{EphemeralContainerCommon: corev1.EphemeralContainerCommon{Name: name}})
		}

		return containers
	}
	terminated := func(name string) corev1.ContainerStatus {
		return corev1.ContainerStatus{Name: name, State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{}}}
	}
	running := func(name string) corev1.ContainerStatus {
		return corev1.ContainerStatus{Name: name, State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}}
	}

	t.Run("none exists", func(t *testing.T) {
		_, found := findUsableDebugContainer(&corev1.Pod{}, "server")
		assert.False(t, found)
	})
	t.Run("containers of other targets and foreign ones are ignored", func(t *testing.T) {
		pod := &corev1.Pod{Spec: corev1.PodSpec{EphemeralContainers: ephemeral("e2e-debug-provisioner-abcde", "debugger-abc", "e2e-debug-server-toolong", "e2e-debug-server")}}
		_, found := findUsableDebugContainer(pod, "server")
		assert.False(t, found)
	})
	t.Run("running one is reused", func(t *testing.T) {
		pod := &corev1.Pod{
			Spec:   corev1.PodSpec{EphemeralContainers: ephemeral("e2e-debug-server-abcde")},
			Status: corev1.PodStatus{EphemeralContainerStatuses: []corev1.ContainerStatus{running("e2e-debug-server-abcde")}},
		}
		name, found := findUsableDebugContainer(pod, "server")
		assert.True(t, found)
		assert.Equal(t, "e2e-debug-server-abcde", name)
	})
	t.Run("one without status yet is reused", func(t *testing.T) {
		pod := &corev1.Pod{Spec: corev1.PodSpec{EphemeralContainers: ephemeral("e2e-debug-server-abcde")}}
		_, found := findUsableDebugContainer(pod, "server")
		assert.True(t, found)
	})
	t.Run("terminated ones are skipped", func(t *testing.T) {
		pod := &corev1.Pod{
			Spec: corev1.PodSpec{EphemeralContainers: ephemeral("e2e-debug-server-aaaaa", "e2e-debug-server-bbbbb")},
			Status: corev1.PodStatus{EphemeralContainerStatuses: []corev1.ContainerStatus{
				terminated("e2e-debug-server-aaaaa"), running("e2e-debug-server-bbbbb"),
			}},
		}
		name, found := findUsableDebugContainer(pod, "server")
		assert.True(t, found)
		assert.Equal(t, "e2e-debug-server-bbbbb", name)

		pod.Status.EphemeralContainerStatuses[1] = terminated("e2e-debug-server-bbbbb")
		_, found = findUsableDebugContainer(pod, "server")
		assert.False(t, found)
	})
}

func newTestPod() *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: testPod, Namespace: testNamespace},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:            "server",
					SecurityContext: &corev1.SecurityContext{Privileged: new(true)},
					VolumeMounts: []corev1.VolumeMount{
						{Name: "data-dir", MountPath: "/data", MountPropagation: ptr.To(corev1.MountPropagationBidirectional)},
						{Name: "kube-api-access", MountPath: serviceAccountMountPath},
					},
				},
			},
		},
	}
}

// withFastPolling makes the tests independent of the real timings.
func withFastPolling(t *testing.T) {
	t.Helper()

	previousInterval, previousTimeout := debugContainerPollInterval, debugContainerStartTimeout
	debugContainerPollInterval, debugContainerStartTimeout = 5*time.Millisecond, 500*time.Millisecond

	t.Cleanup(func() { debugContainerPollInterval, debugContainerStartTimeout = previousInterval, previousTimeout })
}

// reportRunning mimics the kubelet, which starts every ephemeral container that is added to a pod.
func reportRunning(clientset *fake.Clientset) {
	clientset.PrependReactor("update", "pods", func(action clienttesting.Action) (bool, runtime.Object, error) {
		update, ok := action.(clienttesting.UpdateAction)
		if !ok || update.GetSubresource() != "ephemeralcontainers" {
			return false, nil, nil
		}

		pod := update.GetObject().(*corev1.Pod).DeepCopy() //nolint:forcetypeassert
		for _, container := range pod.Spec.EphemeralContainers {
			if findEphemeralContainerStatus(pod, container.Name) == nil {
				pod.Status.EphemeralContainerStatuses = append(pod.Status.EphemeralContainerStatuses, corev1.ContainerStatus{
					Name:  container.Name,
					State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
				})
			}
		}

		return true, pod, clientset.Tracker().Update(corev1.SchemeGroupVersion.WithResource("pods"), pod, pod.Namespace)
	})
}

func TestEnsureDebugContainer(t *testing.T) {
	withFastPolling(t)

	t.Run("adds a container once and reuses it", func(t *testing.T) {
		clientset := fake.NewClientset(newTestPod())
		reportRunning(clientset)

		name, err := ensureDebugContainer(t.Context(), clientset, testNamespace, testPod, "server")
		require.NoError(t, err)
		assert.Regexp(t, `^e2e-debug-server-[a-z0-9]{5}$`, name)

		pod, err := clientset.CoreV1().Pods(testNamespace).Get(t.Context(), testPod, metav1.GetOptions{})
		require.NoError(t, err)
		require.Len(t, pod.Spec.EphemeralContainers, 1)

		debugContainer := pod.Spec.EphemeralContainers[0]
		assert.Equal(t, name, debugContainer.Name)
		assert.Empty(t, debugContainer.TargetContainerName)
		assert.True(t, *debugContainer.SecurityContext.Privileged)
		require.Len(t, debugContainer.VolumeMounts, 1, "service account mount must not be copied")
		assert.Equal(t, "/data", debugContainer.VolumeMounts[0].MountPath)

		again, err := ensureDebugContainer(t.Context(), clientset, testNamespace, testPod, "server")
		require.NoError(t, err)
		assert.Equal(t, name, again)

		pod, err = clientset.CoreV1().Pods(testNamespace).Get(t.Context(), testPod, metav1.GetOptions{})
		require.NoError(t, err)
		assert.Len(t, pod.Spec.EphemeralContainers, 1, "no second container expected")
	})
	t.Run("replaces a terminated container and keeps the old one", func(t *testing.T) {
		pod := newTestPod()
		pod.Spec.EphemeralContainers = []corev1.EphemeralContainer{{EphemeralContainerCommon: corev1.EphemeralContainerCommon{Name: "e2e-debug-server-aaaaa"}}}
		pod.Status.EphemeralContainerStatuses = []corev1.ContainerStatus{{Name: "e2e-debug-server-aaaaa", State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{}}}}
		clientset := fake.NewClientset(pod)
		reportRunning(clientset)

		name, err := ensureDebugContainer(t.Context(), clientset, testNamespace, testPod, "server")
		require.NoError(t, err)
		assert.NotEqual(t, "e2e-debug-server-aaaaa", name)

		updated, err := clientset.CoreV1().Pods(testNamespace).Get(t.Context(), testPod, metav1.GetOptions{})
		require.NoError(t, err)
		assert.Len(t, updated.Spec.EphemeralContainers, 2, "existing ephemeral containers can't be removed")
	})
	t.Run("only regular containers can be targeted", func(t *testing.T) {
		clientset := fake.NewClientset(newTestPod())

		_, err := ensureDebugContainer(t.Context(), clientset, testNamespace, testPod, "csi-init")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not a regular container")
	})
	t.Run("reports a container that doesn't start", func(t *testing.T) {
		clientset := fake.NewClientset(newTestPod()) // nothing reports the container as running

		_, err := ensureDebugContainer(t.Context(), clientset, testNamespace, testPod, "server")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "did not start")
	})
}

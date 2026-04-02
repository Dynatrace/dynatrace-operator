package troubleshoot

import (
	"context"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCheckActiveGateOOMKilled(t *testing.T) {
	dk := testNewDynakubeBuilder(testNamespace, testDynakube).build()

	activeGateLabels := map[string]string{
		k8slabel.AppNameLabel:      k8slabel.ActiveGateComponentLabel,
		k8slabel.AppCreatedByLabel: testDynakube,
		k8slabel.AppManagedByLabel: version.AppName,
	}

	t.Run("no ActiveGate pods found", func(t *testing.T) {
		clt := fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()

		var err error
		logOutput := runWithTestLogger(func(logger logd.Logger) {
			err = checkActiveGates(context.Background(), logger, clt, dk)
		})

		require.NoError(t, err)
		assert.Contains(t, logOutput, "No OOMKilled containers found.")
		assert.NotContains(t, logOutput, "was OOMKilled")
	})

	t.Run("ActiveGate pods found, none OOMKilled", func(t *testing.T) {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dynakube-activegate-0",
				Namespace: testNamespace,
				Labels:    activeGateLabels,
			},
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Name:                 "activegate",
						LastTerminationState: corev1.ContainerState{},
					},
				},
			},
		}

		clt := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(pod).Build()

		var err error
		logOutput := runWithTestLogger(func(logger logd.Logger) {
			err = checkActiveGates(context.Background(), logger, clt, dk)
		})

		require.NoError(t, err)
		assert.Contains(t, logOutput, "No OOMKilled containers found.")
		assert.NotContains(t, logOutput, "was OOMKilled")
	})

	t.Run("ActiveGate pod with OOMKilled container", func(t *testing.T) {
		finishedAt := time.Date(2026, 3, 31, 10, 0, 0, 0, time.UTC)
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dynakube-activegate-0",
				Namespace: testNamespace,
				Labels:    activeGateLabels,
			},
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Name: "activegate",
						LastTerminationState: corev1.ContainerState{
							Terminated: &corev1.ContainerStateTerminated{
								Reason:     "OOMKilled",
								ExitCode:   137,
								FinishedAt: metav1.NewTime(finishedAt),
							},
						},
					},
				},
			},
		}

		clt := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(pod).Build()

		var err error
		logOutput := runWithTestLogger(func(logger logd.Logger) {
			err = checkActiveGates(context.Background(), logger, clt, dk)
		})

		require.NoError(t, err)
		assert.Contains(t, logOutput, "dynakube-activegate-0")
		assert.Contains(t, logOutput, "activegate")
		assert.Contains(t, logOutput, "OOMKilled")
		assert.Contains(t, logOutput, "137")
		assert.NotContains(t, logOutput, "No OOMKilled containers found.")
	})

	t.Run("multiple pods, one OOMKilled", func(t *testing.T) {
		finishedAt := time.Date(2026, 3, 31, 10, 0, 0, 0, time.UTC)
		oomPod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dynakube-activegate-0",
				Namespace: testNamespace,
				Labels:    activeGateLabels,
			},
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Name: "activegate",
						LastTerminationState: corev1.ContainerState{
							Terminated: &corev1.ContainerStateTerminated{
								Reason:     "OOMKilled",
								ExitCode:   137,
								FinishedAt: metav1.NewTime(finishedAt),
							},
						},
					},
				},
			},
		}
		healthyPod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dynakube-activegate-1",
				Namespace: testNamespace,
				Labels:    activeGateLabels,
			},
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Name:                 "activegate",
						LastTerminationState: corev1.ContainerState{},
					},
				},
			},
		}

		clt := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(oomPod, healthyPod).Build()

		var err error
		logOutput := runWithTestLogger(func(logger logd.Logger) {
			err = checkActiveGates(context.Background(), logger, clt, dk)
		})

		require.NoError(t, err)
		assert.Contains(t, logOutput, "dynakube-activegate-0")
		assert.Contains(t, logOutput, "OOMKilled")
		assert.NotContains(t, logOutput, "No OOMKilled containers found.")
	})

	t.Run("pod with non-OOMKilled termination reason", func(t *testing.T) {
		finishedAt := time.Date(2026, 3, 31, 10, 0, 0, 0, time.UTC)
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dynakube-activegate-0",
				Namespace: testNamespace,
				Labels:    activeGateLabels,
			},
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Name: "activegate",
						LastTerminationState: corev1.ContainerState{
							Terminated: &corev1.ContainerStateTerminated{
								Reason:     "Error",
								ExitCode:   1,
								FinishedAt: metav1.NewTime(finishedAt),
							},
						},
					},
				},
			},
		}

		clt := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(pod).Build()

		var err error
		logOutput := runWithTestLogger(func(logger logd.Logger) {
			err = checkActiveGates(context.Background(), logger, clt, dk)
		})

		require.NoError(t, err)
		assert.Contains(t, logOutput, "No OOMKilled containers found.")
		assert.NotContains(t, logOutput, "was OOMKilled")
	})
}

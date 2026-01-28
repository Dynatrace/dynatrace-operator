package pod

import (
	"testing"

	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetWebhookContainerImage(t *testing.T) {
	t.Run("returns image when webhook container is found", func(t *testing.T) {
		webhookPod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "webhook-pod",
				Namespace: "dynatrace",
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "other-container",
						Image: "other-image:latest",
					},
					{
						Name:  dtwebhook.WebhookContainerName,
						Image: "dynatrace/webhook:1.0.0",
					},
				},
			},
		}

		image, err := getWebhookContainerImage(webhookPod)

		require.NoError(t, err)
		assert.Equal(t, "dynatrace/webhook:1.0.0", image)
	})

	t.Run("returns error when webhook container is not found", func(t *testing.T) {
		webhookPod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "webhook-pod",
				Namespace: "dynatrace",
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "other-container",
						Image: "other-image:latest",
					},
					{
						Name:  "another-container",
						Image: "another-image:latest",
					},
				},
			},
		}

		image, err := getWebhookContainerImage(webhookPod)

		require.Error(t, err)
		assert.Empty(t, image)
	})

	t.Run("returns image when webhook container is first in list", func(t *testing.T) {
		webhookPod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "webhook-pod",
				Namespace: "dynatrace",
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  dtwebhook.WebhookContainerName,
						Image: "dynatrace/webhook:2.0.0",
					},
					{
						Name:  "other-container",
						Image: "other-image:latest",
					},
				},
			},
		}

		image, err := getWebhookContainerImage(webhookPod)

		require.NoError(t, err)
		assert.Equal(t, "dynatrace/webhook:2.0.0", image)
	})

	t.Run("returns image when webhook container is last in list", func(t *testing.T) {
		webhookPod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "webhook-pod",
				Namespace: "dynatrace",
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "container1",
						Image: "image1:latest",
					},
					{
						Name:  "container2",
						Image: "image2:latest",
					},
					{
						Name:  dtwebhook.WebhookContainerName,
						Image: "dynatrace/webhook:3.0.0",
					},
				},
			},
		}

		image, err := getWebhookContainerImage(webhookPod)

		require.NoError(t, err)
		assert.Equal(t, "dynatrace/webhook:3.0.0", image)
	})

	t.Run("returns image even if empty string", func(t *testing.T) {
		webhookPod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "webhook-pod",
				Namespace: "dynatrace",
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  dtwebhook.WebhookContainerName,
						Image: "",
					},
				},
			},
		}

		image, err := getWebhookContainerImage(webhookPod)

		require.NoError(t, err)
		assert.Equal(t, "", image)
	})

	t.Run("returns error when pod has no containers", func(t *testing.T) {
		webhookPod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "webhook-pod",
				Namespace: "dynatrace",
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{},
			},
		}

		image, err := getWebhookContainerImage(webhookPod)

		require.Error(t, err)
		assert.Empty(t, image)
	})
}

package k8sevent

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
)

func TestSendCrdVersionMismatch(t *testing.T) {
	t.Run("sends event for DynaKube object", func(t *testing.T) {
		recorder := record.NewFakeRecorder(10)
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-dynakube",
				Namespace: "dynatrace",
			},
		}

		SendCrdVersionMismatch(recorder, dk)

		select {
		case event := <-recorder.Events:
			assert.Contains(t, event, corev1.EventTypeWarning)
			assert.Contains(t, event, crdVersionMismatchReason)
			assert.Contains(t, event, crdVersionMismatchMessage)
		default:
			t.Fatal("Expected event to be recorded, but none was found")
		}
	})

	t.Run("sends event for EdgeConnect object", func(t *testing.T) {
		recorder := record.NewFakeRecorder(10)
		ec := &edgeconnect.EdgeConnect{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-edgeconnect",
				Namespace: "dynatrace",
			},
		}

		SendCrdVersionMismatch(recorder, ec)

		select {
		case event := <-recorder.Events:
			assert.Contains(t, event, corev1.EventTypeWarning)
			assert.Contains(t, event, crdVersionMismatchReason)
			assert.Contains(t, event, crdVersionMismatchMessage)
		default:
			t.Fatal("Expected event to be recorded, but none was found")
		}
	})

	t.Run("works with any client.Object", func(t *testing.T) {
		recorder := record.NewFakeRecorder(10)
		// Use a generic Kubernetes object to ensure interface compatibility
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "default",
			},
		}

		// Should compile and run without errors
		require.NotPanics(t, func() {
			SendCrdVersionMismatch(recorder, pod)
		})

		// Verify event was sent
		select {
		case event := <-recorder.Events:
			assert.Contains(t, event, crdVersionMismatchReason)
		default:
			t.Fatal("Expected event to be recorded, but none was found")
		}
	})
}

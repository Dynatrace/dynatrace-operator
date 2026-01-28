package events

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
)

func TestNewRecorder(t *testing.T) {
	t.Run("creates event recorder with fake recorder", func(t *testing.T) {
		fakeRecorder := record.NewFakeRecorder(10)
		er := NewRecorder(fakeRecorder)

		require.NotNil(t, er)
		assert.Equal(t, fakeRecorder, er.recorder)
	})
}

func TestEventRecorder_Setup(t *testing.T) {
	t.Run("sets dynakube and pod from mutation request", func(t *testing.T) {
		fakeRecorder := record.NewFakeRecorder(10)
		er := NewRecorder(fakeRecorder)

		dk := dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-dk",
				Namespace: "dynatrace",
			},
		}
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "test-ns",
			},
		}

		mutationRequest := &dtwebhook.MutationRequest{
			BaseRequest: &dtwebhook.BaseRequest{
				Pod:      pod,
				DynaKube: dk,
			},
			Context: context.Background(),
		}

		er.Setup(mutationRequest)

		assert.Equal(t, &dk, er.dk)
		assert.Equal(t, pod, er.pod)
	})
}

func TestEventRecorder_SendPodInjectEvent(t *testing.T) {
	t.Run("sends inject event with correct parameters", func(t *testing.T) {
		fakeRecorder := record.NewFakeRecorder(10)
		er := NewRecorder(fakeRecorder)

		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-dk",
				Namespace: "dynatrace",
			},
		}
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-pod-",
				Namespace:    "test-ns",
			},
		}

		er.dk = dk
		er.pod = pod

		er.SendPodInjectEvent()

		select {
		case event := <-fakeRecorder.Events:
			assert.Contains(t, event, "Normal")
			assert.Contains(t, event, "Inject")
			assert.Contains(t, event, "test-pod-")
			assert.Contains(t, event, "test-ns")
			assert.Contains(t, event, "Injecting the necessary info into pod")
		default:
			t.Fatal("Expected an event to be recorded")
		}
	})

	t.Run("uses generate name in event message", func(t *testing.T) {
		fakeRecorder := record.NewFakeRecorder(10)
		er := NewRecorder(fakeRecorder)

		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-dk",
				Namespace: "dynatrace",
			},
		}
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "my-app-deployment-",
				Namespace:    "production",
			},
		}

		er.dk = dk
		er.pod = pod

		er.SendPodInjectEvent()

		select {
		case event := <-fakeRecorder.Events:
			assert.Contains(t, event, "my-app-deployment-")
			assert.Contains(t, event, "production")
		default:
			t.Fatal("Expected an event to be recorded")
		}
	})
}

func TestEventRecorder_SendPodUpdateEvent(t *testing.T) {
	t.Run("sends update event with correct parameters", func(t *testing.T) {
		fakeRecorder := record.NewFakeRecorder(10)
		er := NewRecorder(fakeRecorder)

		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-dk",
				Namespace: "dynatrace",
			},
		}
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-pod-",
				Namespace:    "test-ns",
			},
		}

		er.dk = dk
		er.pod = pod

		er.SendPodUpdateEvent()

		select {
		case event := <-fakeRecorder.Events:
			assert.Contains(t, event, "Normal")
			assert.Contains(t, event, "UpdatePod")
			assert.Contains(t, event, "test-pod-")
			assert.Contains(t, event, "test-ns")
			assert.Contains(t, event, "Updating pod")
			assert.Contains(t, event, "missing containers")
		default:
			t.Fatal("Expected an event to be recorded")
		}
	})
}

func TestEventRecorder_SendMissingDynaKubeEvent(t *testing.T) {
	t.Run("sends missing dynakube event with correct parameters", func(t *testing.T) {
		fakeRecorder := record.NewFakeRecorder(10)
		er := NewRecorder(fakeRecorder)

		er.SendMissingDynaKubeEvent("test-namespace", "test-dynakube")

		select {
		case event := <-fakeRecorder.Events:
			assert.Contains(t, event, "Warning")
			assert.Contains(t, event, "MissingDynakube")
			assert.Contains(t, event, "test-namespace")
			assert.Contains(t, event, "test-dynakube")
			assert.Contains(t, event, "assigned to DynaKube instance")
			assert.Contains(t, event, "doesn't exist")
		default:
			t.Fatal("Expected an event to be recorded")
		}
	})

	t.Run("creates temporary dynakube object for event", func(t *testing.T) {
		fakeRecorder := record.NewFakeRecorder(10)
		er := NewRecorder(fakeRecorder)

		namespaceName := "my-namespace"
		dynakubeName := "my-dynakube"

		er.SendMissingDynaKubeEvent(namespaceName, dynakubeName)

		select {
		case event := <-fakeRecorder.Events:
			// Verify event was sent with the correct namespace and dynakube names
			assert.Contains(t, event, namespaceName)
			assert.Contains(t, event, dynakubeName)
		default:
			t.Fatal("Expected an event to be recorded")
		}
	})

	t.Run("sends warning event type", func(t *testing.T) {
		fakeRecorder := record.NewFakeRecorder(10)
		er := NewRecorder(fakeRecorder)

		er.SendMissingDynaKubeEvent("test-ns", "test-dk")

		select {
		case event := <-fakeRecorder.Events:
			// Verify it's a Warning event (not Normal)
			assert.Contains(t, event, "Warning")
			assert.NotContains(t, event, "Normal")
		default:
			t.Fatal("Expected an event to be recorded")
		}
	})
}

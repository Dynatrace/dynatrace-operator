package events

import (
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/events"
)

const (
	injectEvent          = "Inject"
	updatePodEvent       = "UpdatePod"
	missingDynakubeEvent = "MissingDynakube"
)

// EventRecorder is a type alias to not have to import both tools/events and this package.
type EventRecorder = events.EventRecorder

func SendPodInjectEvent(recorder events.EventRecorder, dk *dynakube.DynaKube, pod *corev1.Pod) {
	msg := fmt.Sprintf("Injecting the necessary info into pod %s in namespace %s", pod.GenerateName, pod.Namespace)
	recorder.Eventf(dk,
		nil, // no related obj
		corev1.EventTypeNormal,
		injectEvent,
		msg,
		msg)
}

func SendPodUpdateEvent(recorder events.EventRecorder, dk *dynakube.DynaKube, pod *corev1.Pod) {
	msg := fmt.Sprintf("Updating pod %s in namespace %s with missing containers", pod.GenerateName, pod.Namespace)
	recorder.Eventf(dk,
		nil, // no related obj
		corev1.EventTypeNormal,
		updatePodEvent,
		msg,
		msg)
}

func SendMissingDynaKubeEvent(recorder events.EventRecorder, namespaceName, dynakubeName string) {
	msg := fmt.Sprintf("Namespace '%s' is assigned to DynaKube instance '%s' but this instance doesn't exist", namespaceName, dynakubeName)
	recorder.Eventf(
		&dynakube.DynaKube{ObjectMeta: metav1.ObjectMeta{Name: dynakubeName, Namespace: namespaceName}},
		nil, // no related obj
		corev1.EventTypeWarning,
		missingDynakubeEvent,
		msg,
		msg)
}

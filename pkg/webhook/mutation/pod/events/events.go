package events

import (
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/events"
)

const (
	injectEvent          = "Inject"
	updatePodEvent       = "UpdatePod"
	missingDynakubeEvent = "MissingDynakube"
)

type EventRecorder struct {
	dk       *dynakube.DynaKube
	pod      *corev1.Pod
	recorder events.EventRecorder
}

func NewRecorder(recorder events.EventRecorder) EventRecorder {
	return EventRecorder{recorder: recorder}
}

func (er *EventRecorder) Setup(mutationRequest *dtwebhook.MutationRequest) {
	er.dk = &mutationRequest.DynaKube
	er.pod = mutationRequest.Pod
}

func (er *EventRecorder) SendPodInjectEvent() {
	msg := fmt.Sprintf("Injecting the necessary info into pod %s in namespace %s", er.pod.GenerateName, er.pod.Namespace)
	er.recorder.Eventf(er.dk,
		nil, // no related obj
		corev1.EventTypeNormal,
		injectEvent,
		msg,
		msg)
}

func (er *EventRecorder) SendPodUpdateEvent() {
	msg := fmt.Sprintf("Updating pod %s in namespace %s with missing containers", er.pod.GenerateName, er.pod.Namespace)
	er.recorder.Eventf(er.dk,
		nil, // no related obj
		corev1.EventTypeNormal,
		updatePodEvent,
		msg,
		msg)
}

func (er *EventRecorder) SendMissingDynaKubeEvent(namespaceName, dynakubeName string) {
	msg := fmt.Sprintf("Namespace '%s' is assigned to DynaKube instance '%s' but this instance doesn't exist", namespaceName, dynakubeName)
	er.recorder.Eventf(
		&dynakube.DynaKube{ObjectMeta: metav1.ObjectMeta{Name: dynakubeName, Namespace: namespaceName}},
		nil, // no related obj
		corev1.EventTypeWarning,
		missingDynakubeEvent,
		msg,
		msg)
}

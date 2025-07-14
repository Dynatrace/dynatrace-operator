package events

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/common"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
)

const (
	injectEvent          = "Inject"
	updatePodEvent       = "UpdatePod"
	IncompatibleCRDEvent = "IncompatibleCRDPresent"
	missingDynakubeEvent = "MissingDynakube"
)

type EventRecorder struct {
	dk       *dynakube.DynaKube
	pod      *corev1.Pod
	recorder record.EventRecorder
}

func NewRecorder(recorder record.EventRecorder) EventRecorder {
	return EventRecorder{recorder: recorder}
}

func (er *EventRecorder) Setup(mutationRequest *dtwebhook.MutationRequest) {
	er.dk = &mutationRequest.DynaKube
	er.pod = mutationRequest.Pod
}

func (er *EventRecorder) SendPodInjectEvent() {
	er.recorder.Eventf(er.dk,
		corev1.EventTypeNormal,
		injectEvent,
		"Injecting the necessary info into pod %s in namespace %s", er.pod.GenerateName, er.pod.Namespace)
}

func (er *EventRecorder) SendPodUpdateEvent() {
	er.recorder.Eventf(er.dk,
		corev1.EventTypeNormal,
		updatePodEvent,
		"Updating pod %s in namespace %s with missing containers", er.pod.GenerateName, er.pod.Namespace)
}

func (er *EventRecorder) SendMissingDynaKubeEvent(namespaceName, dynakubeName string) {
	template := "Namespace '%s' is assigned to DynaKube instance '%s' but this instance doesn't exist"
	er.recorder.Eventf(
		&dynakube.DynaKube{ObjectMeta: metav1.ObjectMeta{Name: dynakubeName, Namespace: namespaceName}},
		corev1.EventTypeWarning,
		missingDynakubeEvent,
		template, namespaceName, dynakubeName)
}

func (er *EventRecorder) SendOneAgentAPMWarningEvent(webhookPod *corev1.Pod) {
	er.recorder.Event(webhookPod,
		corev1.EventTypeWarning,
		IncompatibleCRDEvent,
		"Unsupported OneAgentAPM CRD still present in cluster, please remove to proceed")
}
